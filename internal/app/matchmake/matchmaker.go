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
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"golang.org/x/sync/errgroup"
)

type Matchmaker struct {
	TransportBus               *TransportBus
	RoomRepository             *room.Repository
	PlayerRepository           *players.Repository
	MatchmakingTaskCoordinator *taskcoordinator.MatchmakingTaskCoordinator
}

const (
	numWorkers              = 4
	WorkerRetryIntervalSecs = 5 //seconds
	RetryLockPauseTime      = time.Millisecond * 100
)

// Start begins processing incoming matchmaking messages from clients and spawns <numWorkers> go routines to
// begin pulling pending matchmaking tasks and attempt to make new matches. Errors produced by Processing matchmaking requests
// or from the matchmaking loop should be breaking errors that cause the entire matchmaker service to go down.
// TODO dig through all error sources and create handle-able errors to be checked within the functions called in the
//
//	go routines.
func (m *Matchmaker) Start(ctx context.Context) {
	var eg errgroup.Group

	eg.Go(func() error {
		return m.ProcessMatchmakingMessages(ctx)
	})

	for i := 0; i < numWorkers; i++ {
		eg.Go(func() error {
			return m.runMatchmakingLoop(ctx, i)
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

func (m *Matchmaker) runMatchmakingLoop(ctx context.Context, workerOffset int) error {
	alertCh, errCh := m.TransportBus.ListenForMatchmakeWorkerNotifications(ctx)

	<-time.NewTimer((time.Millisecond * 25) * time.Duration(workerOffset)).C //allow some time to offset workers

	timer := time.NewTicker(WorkerRetryIntervalSecs * time.Second)
	idleCount := 0

	tryMakingMatches := func() error {
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
			err := tryMakingMatches()
			if utils.ErrorsIsAny(err, taskcoordinator.NoPendingTasksAvailableErr, room.ErrDidNotLock) {
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
			if utils.ErrorsIsAny(err, taskcoordinator.NoPendingTasksAvailableErr, room.ErrDidNotLock) {
				continue
			}
			if err != nil {
				log.Printf("encountered an error claiming next pending task - %v\n", err)
			}

		case err := <-errCh:
			//TODO if err is fatal, return it, shouldn't be limited to context.Canceled
			log.Printf("worker received error - %v\n", err)
			if errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

// ProcessMatchmakingMessages
// TODO Errors produced in this method need to be examined (through testing or other) in order to create handle-able errors
//
//	that do not cause this method to return an error, but handle them gracefully and continue processing.
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
					err = m.processExitMatchmakingRequest(gCtx, msg) //errors received from this are breaking
					if err != nil {
						return err
					}
				case *RequestMatchmakingMessage:
					err = m.processMatchmakingRequest(gCtx, msg) // errors received from this are breaking
					if err != nil {
						return err
					}
				}
				//TODO maybe handle breaking errors that should cancel anymore message processing (any failures to interact with redis db.
				//TODO 	otherwise should just continue reading from the server message stream
				if err == nil {
					err = matchmakingMsg.Ack(gCtx)
					//err = m.TransportBus.AckServerMessage(gCtx, matchmakingMsg.GetID())
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

	if len(openRooms) == 0 {
		err = m.MatchmakingTaskCoordinator.MoveInProgressTaskToPending(ctx, rm.RoomId)
		if err != nil {
			return err
		}
		return nil
	}

	didMakeMatch := false
	for _, openRm := range openRooms {
		combinedRoom, err := m.combineRooms(ctx, rm, openRm)
		if err != nil {
			if utils.ErrorsIsAny(err, room.ErrRoomFull, room.ErrDidNotLock) {
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

func (m *Matchmaker) combineRooms(ctx context.Context, rm room.Room, rmToJoin room.Room) (room.Room, error) {
	var combinedRoom room.Room

	lockKey, err := m.RoomRepository.LockRoom(ctx, rmToJoin.RoomId)
	if err != nil {
		return combinedRoom, err
	}
	defer func() {
		err = m.RoomRepository.UnlockRoom(ctx, rmToJoin.RoomId, lockKey)
		if err != nil {
			log.Println("[VERY BAD SHOULD NOT HAPPEN] failed to unlock room when trying to combine rooms - ", err)
		}
	}()

	combinedRoom, err = m.RoomRepository.CombineRooms(ctx, rm.RoomId, rmToJoin.RoomId) //this means that "rm" gets absorbed into "openRm",
	if err != nil {
		return combinedRoom, err
	}
	err = m.MatchmakingTaskCoordinator.RemoveInProgressTask(ctx, rm.RoomId) //room is absorbed, no more need for tasks
	if err != nil {
		return combinedRoom, err
	}

	return combinedRoom, nil
}

// processExitMatchmakingRequest is executed after the websocket server sends an ExitMatchmakingRequest. The websocket
// server sends this after the client disconnects during the matchmaking phase, whether done by the user or another issue.
// TODO Errors produced in this method need to be examined (through testing or other) in order to create handle-able errors
//
//	that do not cause this method to return an error, but handle them gracefully and continue processing.
func (m *Matchmaker) processExitMatchmakingRequest(ctx context.Context, msg *ExitMatchmakingMessage) error {
	var lockKey, roomId uuidstring.ID
	var err error

	for {
		roomId, err = m.PlayerRepository.GetPlayerRoom(ctx, msg.UserId)
		if err != nil {
			return err
		}
		lockKey, err = m.RoomRepository.LockRoom(ctx, roomId)
		if err == nil {
			break
		} else {
			if !errors.Is(err, room.ErrDidNotLock) {
				return err
			}
		}
		<-time.NewTimer(RetryLockPauseTime).C
	}

	defer func() {
		err := m.RoomRepository.UnlockRoom(ctx, roomId, lockKey)
		if err != nil {
			log.Printf("failed to unlock room[%s] - %v\n", roomId, err)
		}
	}()
	newPlayerIds, err := m.RoomRepository.RemovePlayer(ctx, roomId, msg.UserId, msg.UserSkill)
	if err != nil {
		return err
	}

	for _, playerId := range newPlayerIds {
		playerLeftMsg := NewPlayerLeftRoomMessage(msg.UserId)
		fmt.Printf("sending message to player[%s] - %v\n", playerId, playerLeftMsg)
		err = m.TransportBus.SendToClient(ctx, playerId, &playerLeftMsg)
		if err != nil {
			log.Printf("failed to send message to client[%s] - %v\n", playerId, err)
		}
	}

	err = m.MatchmakingTaskCoordinator.RemovePendingTask(ctx, roomId)
	if err != nil {
		log.Println("error removing pending task")
	}

	err = m.PlayerRepository.SetPlayerInactive(ctx, msg.UserId)
	if err != nil {
		log.Println("error setting player[", msg.UserId, "] inactive - ", err)
	}
	return nil
}
