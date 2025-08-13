package matchmake

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/constants"
	"github.com/bkohler93/game-backend/internal/shared/players"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"golang.org/x/sync/errgroup"
)

type Matchmaker struct {
	//MatchmakingClientMessageProducer transport.DynamicMessageProducer
	//MatchmakingServerMessageConsumer transport.MessageGroupConsumer
	TransportBus               *TransportBus
	RoomRepository             *room.Repository
	PlayerRepository           *players.Repository
	MatchmakingTaskCoordinator *taskcoordinator.MatchmakingTaskCoordinator
}

const (
	numWorkers              = 4
	WorkerRetryIntervalSecs = 5 //seconds
)

func (m *Matchmaker) Start(ctx context.Context) {
	err := m.RoomRepository.CreateRoomIndex(ctx)
	if err != nil {
		log.Printf("failed to create room index. Shutting down - %v\n", err)
		return
	}
	var eg errgroup.Group
	eg.Go(func() error {
		return m.ProcessMatchmakingMessages(ctx)
	})

	for i := 0; i < numWorkers; i++ {
		eg.Go(func() error {
			return m.runMatchmakingLoop(ctx)
		})
	}

	log.Println("ready to process matchmaking requests")
	if err := eg.Wait(); err != nil {
		log.Printf("shutting down due to error - %v\n", err)
	} else {
		log.Println("shutting down gracefully")
	}
}

func (m *Matchmaker) processMatchmakingRequest(ctx context.Context, req *RequestMatchmakingMessage) error {
	rm := room.Room{
		RoomId:            uuidstring.NewID(),
		PlayerCount:       1,
		AverageSkill:      req.Skill,
		Region:            req.Region,
		PlayerIds:         []uuidstring.ID{req.UserId},
		CreatedAt:         time.Now().Unix(),
		MatchmakeAttempts: 0,
		IsFull:            0,
	}
	err := m.RoomRepository.CreateRoom(ctx, rm)
	if err != nil {
		log.Printf("error creating new room - %v\n", err)
	}
	msg := NewRoomChangedMessage(rm.RoomId, rm.PlayerCount, rm.AverageSkill)

	err = m.TransportBus.SendToClient(ctx, req.UserId, msg)
	if err != nil {
		return err
	}
	err = m.PlayerRepository.SetPlayerActive(ctx, req.UserId, rm.RoomId)
	if err != nil {
		log.Printf("error storing player in active player list - %v\n", err)
	}
	err = m.MatchmakingTaskCoordinator.AddPendingTask(ctx, rm.RoomId, time.Now().Add(room.PendingTime).Unix())
	if err != nil {
		log.Printf("error adding room matchmaking task 0 %v\n", err)
	}

	err = m.TransportBus.NotifyMatchmakeWorkers(ctx)
	if err != nil {
		log.Printf("error notifying matchmaking task workers - %v\n", err)
	}
	return nil
}

func (m *Matchmaker) runMatchmakingLoop(ctx context.Context) error {
	alertCh, errCh := m.TransportBus.ListenForMatchmakeWorkerNotifications(ctx)
	timer := time.NewTicker(WorkerRetryIntervalSecs * time.Second)
	idleCount := 0

	tryMakingMatches := func() error {
		fmt.Println("checking for a pending task")
		roomId, err := m.MatchmakingTaskCoordinator.ClaimNextPendingTask(ctx)
		if err != nil {
			return err
		}
		err = m.attemptMatchmake(ctx, roomId)
		if err != nil {
			return err
		}
		return nil
	}

	for {
		select {
		case <-timer.C:
			fmt.Println("timer trigger matchmake retry - idleCount", idleCount)
			err := tryMakingMatches()
			if utils.ErrorsAny(err, taskcoordinator.NoPendingTasksAvailableErr, room.ErrDidNotLock) {
				fmt.Println("no pending tasks avaialable")
				idleCount++
				if idleCount > 5 {
					idleCount = 5
				}
				newDuration := time.Duration(WorkerRetryIntervalSecs*idleCount) * time.Second
				timer.Reset(newDuration)
				continue
			}
			if err != nil {
				log.Printf("encountered an error claiming next pending task - %v\n", err)
			} else {
				idleCount = 0 //successfully found pending task and tried to matchmake, reset idle time
			}

		case alert := <-alertCh:
			idleCount = 0
			if alert != "" {
				log.Println("received unknown alert from notify matchmake workers channel,", alert)
				continue
			}
			err := tryMakingMatches()
			if utils.ErrorsAny(err, taskcoordinator.NoPendingTasksAvailableErr, room.ErrDidNotLock) {
				continue
			}
			if err != nil {
				log.Printf("encountered an error claiming next pending task - %v\n", err)
			}

		case err := <-errCh:
			//TODO if err is fatal, return it
			log.Printf("worker received error - %v\n", err)
			if errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

func (m *Matchmaker) ProcessMatchmakingMessages(ctx context.Context) error {
	msgCh, errCh := m.TransportBus.StartReceivingServerMessages(ctx)
	eg, gCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		for {
			select {
			case <-gCtx.Done():
				return nil
			case matchmakingMsg, ok := <-msgCh:
				if !ok {
					return nil
				}
				var err error
				switch msg := matchmakingMsg.(type) {
				case *ExitMatchmakingMessage:
					//TODO err = m.processExitMatchmakingRequest(gCtx, msg)
					err = nil
				case *RequestMatchmakingMessage:
					err = m.processMatchmakingRequest(gCtx, msg)
					if err != nil {
						return err
					}
				}
				//TODO handle breaking errors that should cancel anymore message processing (any failures to interact with redis db.
				//TODO 	otherwise should just continue reading from the server message stream
				if err == nil {
					err = m.TransportBus.AckServerMessage(gCtx, matchmakingMsg.GetID())
					if errors.Is(err, transport.ErrScriptNotFound) {
						return err
					}
				}
				if err != nil {
					log.Printf("received an error - %v", err)
				}
			}
		}
	})

	eg.Go(func() error {
		select {
		case <-gCtx.Done():
			return nil
		case err := <-errCh:
			return err
		}
	})

	return eg.Wait()
}

func (m *Matchmaker) attemptMatchmake(ctx context.Context, roomId uuidstring.ID) error {
	lockKey, err := m.RoomRepository.LockRoom(ctx, roomId)
	if err != nil {
		return err
	}
	defer func() {
		err = m.RoomRepository.UnlockRoom(ctx, roomId, lockKey)
		if err != nil {
			log.Println("failed to unlock room -", err)
		}
	}()

	rm, err := m.RoomRepository.GetRoom(ctx, roomId)
	if err != nil {
		return err
	}
	openRooms, err := m.RoomRepository.QueryOpenRooms(ctx, &room.QueryRoomRequest{
		RoomId:              roomId,
		Skill:               rm.AverageSkill,
		TimeCreated:         rm.CreatedAt,
		Region:              rm.Region,
		MaxAllowablePlayers: constants.MaxPlayerCount - rm.PlayerCount,
	})
	if err != nil {
		return err
	}
	fmt.Println("received open rooms", openRooms)

	if len(openRooms) == 0 {
		fmt.Println("there are no open rooms found")
		err = m.MatchmakingTaskCoordinator.MoveInProgressTaskToPending(ctx, rm.RoomId)
		if err != nil {
			return err
		}
		return nil
	}

	didMakeMatch := false
	for _, openRm := range openRooms {
		combinedRoom, err := m.didMakeMatch(ctx, rm, openRm)
		if err != nil {
			if errors.Is(err, room.ErrRoomFull) {
				continue
			}
			return err
		}
		didMakeMatch = true
		var clientMsg MatchmakingClientMessage
		if combinedRoom.PlayerCount == constants.MaxPlayerCount {
			err = m.MatchmakingTaskCoordinator.RemovePendingTask(ctx, combinedRoom.RoomId) // room is full no more need for tasks
			clientMsg = NewRoomFullMessage(combinedRoom.RoomId, combinedRoom.PlayerCount)
		} else {
			clientMsg = NewRoomChangedMessage(combinedRoom.RoomId, combinedRoom.PlayerCount, combinedRoom.AverageSkill)
		}

		for _, playerId := range combinedRoom.PlayerIds {
			err = m.TransportBus.SendToClient(ctx, playerId, clientMsg)
			if err != nil {
				log.Printf("encountered an error sending message to client - %v", err)
			}
		}
	}

	if !didMakeMatch {
		err = m.MatchmakingTaskCoordinator.MoveInProgressTaskToPending(ctx, rm.RoomId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Matchmaker) didMakeMatch(ctx context.Context, rm room.Room, rmToJoin room.Room) (room.Room, error) {
	var combinedRoom room.Room
	rmToJoinKeyLock, err := m.RoomRepository.LockRoom(ctx, rmToJoin.RoomId)
	if err != nil {
		return combinedRoom, err
	}
	defer func() {
		err = m.RoomRepository.UnlockRoom(ctx, rmToJoin.RoomId, rmToJoinKeyLock)
		if err != nil {
			log.Println("failed to unlock room in didMakeMatch -", err)
		}
	}()
	combinedRoom, err = m.RoomRepository.CombineRooms(ctx, rm, rmToJoin) //this means that "rm" gets absorbed into "openRm",
	if err != nil {
		//TODO tolerable errors include openRm is full/would be overpopulated, try to join next room
		return combinedRoom, err //this should be unrecoverable, crashing the workers
	}
	err = m.MatchmakingTaskCoordinator.RemoveInProgressTask(ctx, rm.RoomId) //room is absorbed, no more need for tasks
	if err != nil {
		//TODO This error is not good if it occurs. Important to note if this ever happens
		log.Printf("=== terrible error === unknown error removing room with id[%s] from inprogress and pending tasks - %v", rm.RoomId, err)
		return combinedRoom, err
	}

	return combinedRoom, nil
}

//func canMatch(u1 message.MatchmakingRequest, u2 message.MatchmakingRequest) bool {
//	//TODO make the match rule more... something
//	return u1.UserId != u2.UserId
//}
