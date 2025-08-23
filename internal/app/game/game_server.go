package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	StartGamePeriod = time.Second * 5
)

type GameServer struct {
	GameTaskCoordinator *taskcoordinator.GameTaskCoordinator
	roomStore           room.Store
	transportFactory    *TransportFactory
}

func New(gtc *taskcoordinator.GameTaskCoordinator, rs room.Store, tf *TransportFactory) *GameServer {
	return &GameServer{
		GameTaskCoordinator: gtc,
		roomStore:           rs,
		transportFactory:    tf,
	}
}

func (gs *GameServer) Start(ctx context.Context) {
	t := time.NewTicker(StartGamePeriod)
	defer t.Stop()

	var wg sync.WaitGroup

	log.Println("started listening for new game tasks")
	for {
		select {
		case <-t.C:
			roomID, err := gs.GameTaskCoordinator.DequeueNewGameTask(ctx)
			if err != nil {
				if !errors.Is(err, taskcoordinator.ErrNoTasksAvailable) {
					log.Printf("encountered an error dequeueing new game task - %v", err)
				}
				continue
			}
			gameCtx, cancel := context.WithCancel(ctx)

			fmt.Printf("received new game task for room[%s]\n", roomID)
			wg.Add(1)

			go func() {
				defer cancel()
				err := gs.StartGame(gameCtx, roomID)
				if err != nil {
					log.Printf("game server encountered an error - %v", err)
				}
				wg.Done()
			}()
		case <-ctx.Done():
			wg.Wait()
			return
		}
	}
}

func (gs *GameServer) StartGame(ctx context.Context, roomID uuidstring.ID) error {
	rm, err := gs.roomStore.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	playerIds := rm.PlayerIds
	playersConnected := make(map[uuidstring.ID]bool)
	for _, id := range playerIds {
		playersConnected[id] = false
	}
	consumer, err := gs.transportFactory.GameServerMsgConsumerBuilder(ctx, roomID.String())
	if err != nil {
		return err
	}

	consumerCtx, cancel := context.WithCancel(ctx)
	receiveCh, errCh := consumer.StartReceiving(consumerCtx)
	for {
		select {
		case <-ctx.Done():
			cancel()
			return nil
		case envCtx := <-receiveCh:
			var confirmMatchMsg *ConfirmMatchMessage
			err = json.Unmarshal(envCtx.Env.Payload, &confirmMatchMsg)
			if err != nil {
				log.Printf("failed to unmarshal envelope payload - %v\n", err)
				continue
			}
			playersConnected[confirmMatchMsg.UserId] = true
			if allPlayersConnected(&playersConnected) {
				cancel() //stop listening for game server messages, will start listening in game
				g := NewGame(roomID, gs.transportFactory, rm.PlayerIds)
				return g.Start(ctx)
			}
		case err := <-errCh:
			log.Printf("encountered an error from ServerMessage stream - %v\n", err)
		}
	}
}

func allPlayersConnected(playersConnected *map[uuidstring.ID]bool) bool {
	for _, isConnected := range *playersConnected {
		if !isConnected {
			return false
		}
	}
	return true
}
