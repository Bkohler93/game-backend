package game

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	StartGamePeriod = time.Millisecond * 100
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

			wg.Add(1)

			go func() {
				gameCtx, cancel := context.WithCancel(ctx)
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

	messageConsumer, err := gs.transportFactory.GameServerMsgConsumerBuilder(ctx, roomID.String())
	if err != nil {
		return err
	}
	messageProducer := gs.transportFactory.GameClientMsgProducerBuilder()
	bus := NewGameTransportBus(messageConsumer, messageProducer)

	g := NewGame(ctx, roomID, bus, rm.PlayerIds)
	return g.Start()
}
