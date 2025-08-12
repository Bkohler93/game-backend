package matchmake

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/players"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/internal/shared/transport"
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
	numWorkers = 4
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
			return m.makeMatches(ctx)
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
	queryRoomReq := room.QueryRoomRequest{
		Skill:       req.Skill,
		TimeCreated: req.TimeCreated,
		Region:      req.Region,
	}
	openRooms, err := m.RoomRepository.QueryOpenRooms(ctx, queryRoomReq)
	if err != nil {
		return err
	}
	roomFound := false
	for _, openRoom := range openRooms {
		joinRoomRequest := room.JoinRoomRequest{
			UserId: req.UserId,
			Skill:  req.Skill,
		}
		rm, err := m.RoomRepository.JoinRoom(ctx, joinRoomRequest, openRoom)
		if err != nil {
			if err.Error() == "ROOM_FULL" {
				continue
			}
			log.Printf("failed attempt to join the open room - %v\n", err)
			continue
		}
		roomFound = true

		//TODO rm.IsFull might be true, which case you should send RoomFullMessage to all member of room.
		//TODO AFTER THIS happens, clients need to stop listening on matchmaking channel and move to
		//TODO listening on game channel. Each game is associated with the roomID, so matchmaker should
		//TODO also be in communication with the game server that a game in room[roomID] is starting.
		for _, playerId := range rm.PlayerIds {
			if playerId == req.UserId {
				var msg MatchmakingClientMessage
				if rm.IsFull == 1 {
					msg = NewRoomFullMessage(rm.RoomId, rm.PlayerCount)
				} else {
					msg = NewRoomChangedMessage(rm.RoomId, rm.PlayerCount, rm.AverageSkill)
				}
				bytes, err := json.Marshal(msg)
				if err != nil {
					log.Printf("failed to marshal object into json -%v\n", err)
				}
				err = m.TransportBus.SendToClient(ctx, playerId, msg)
				if err != nil {
					log.Printf("error sending %v to client\n", bytes)
				}
				err = m.PlayerRepository.SetPlayerActive(ctx, playerId, rm.RoomId)
				if err != nil {
					log.Printf("error storing player in active player list - %v\n", err)
				}
			} else {
				var msg MatchmakingClientMessage
				if rm.IsFull == 1 {
					msg = NewRoomFullMessage(rm.RoomId, rm.PlayerCount)
				} else {
					msg = NewPlayerJoinedRoomMessage(req.UserId)
				}
				err = m.TransportBus.SendToClient(ctx, playerId, msg)
				if err != nil {
					log.Printf("error publishing %v to client\n", msg)
				}
			}

			break
		}
	}

	if !roomFound {
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
		err = m.RoomRepository.CreateRoom(ctx, rm)
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

		//TODO Publish to PubSub 'matchmake:notify_workers'
		err = m.TransportBus.NotifyMatchmakeWorkers(ctx)
		if err != nil {
			log.Printf("error notifying matchmaking task workers - %v\n", err)
		}
	}
	return nil
}

func (m *Matchmaker) makeMatches(ctx context.Context) error {
	// TODO:
	//1. Poll `matchmake_tasks` for the oldest task in the set
	//2. Atomically remove it from `matchmake_tasks`, insert into `matchmake_inprogress` with current time as score.
	//3. Query for compatible rooms (region/skill/time_created to adjust skill further).
	//4. If a match is found:
	//    - Perform atomic room combine (player transfer, skill/count update).
	//    - Remove affected room IDs from both queues.
	//    - Send results to `matchmake:client_message:<client_id>`. Could be either types `room_full` or `room_combined` with an updated playercount.
	//5. If no match:
	//    - Update retry info (in room or hash).
	//    - Compute new retry time and move back to `matchmake_tasks`.
	//
	//If idle, workers block on pubsub `matchmake:notify_workers`, which is published to anytime an inactive `matchmake_inprogress` is detected or a room is created after an unsuccessful initial matchmake attempt.
	return nil
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

//func canMatch(u1 message.MatchmakingRequest, u2 message.MatchmakingRequest) bool {
//	//TODO make the match rule more... something
//	return u1.UserId != u2.UserId
//}
