package matchmake

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/players"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"golang.org/x/sync/errgroup"
)

type Matchmaker struct {
	MatchmakingClientMessageProducer MatchmakingClientMessageProducer
	RoomRepository                   *room.Repository
	PlayerRepository                 *players.Repository
	MatchmakingTaskCoordinator       *taskcoordinator.MatchmakingTaskCoordinator
}

const (
	numWorkers = 4
)

func (m *Matchmaker) Start(ctx context.Context) {
	var eg errgroup.Group
	eg.Go(func() error {
		return m.HandleMatchmakeRequests(ctx)
	})

	for i := 0; i < numWorkers; i++ {
		eg.Go(func() error {
			return m.makeMatches(ctx)
		})
	}

	if err := eg.Wait(); err != nil {
		fmt.Printf("Matchmaker cancelling due to error - %v", err)
	}
	//for {
	//	fmt.Println("Listening for new Matchmaking messages.")
	//
	//	var msg message.BaseMatchmakingServerMessage
	//	//stream := rediskeys.MatchmakingServerMessageStream
	//	//err := m.MessageBus.StartConsuming(ctx, stream, &msg)
	//	//if err != nil {
	//	//	fmt.Printf("failed to consume from %s - %v\n", stream, err)
	//	//	continue
	//	//}
	//	switch msg.Type {
	//	case message.ServerMessageTypeMatchmakingRequest:
	//		var req message.MatchmakingRequest
	//		err := json.Unmarshal(msg.Payload, &req)
	//		if err != nil {
	//			fmt.Printf("Error unmarshalling matchmaking request: %v\n", err)
	//			continue
	//		}
	//		go m.processMatchmakingRequest(ctx, req)
	//		break
	//	case message.ServerMessageTypeMatchmakingExit:
	//	}
	//	// entries, err := m.rdb.XRead(ctx, &goredis.XReadArgs{
	//	// 	Streams: []string{"matchmake:request", "$"},
	//	// 	Count:   1,
	//	// 	Block:   0,
	//	// }).Result()
	//	// if err != nil {
	//	// 	fmt.Printf("failed to read from matchmake:request stream - %v\n", err)
	//	// 	continue
	//	// }
	//	// res := entries[0].Messages[0].Values
	//	// var req MatchRequest
	//	// err = interfacestruct.Structify(res, &req)
	//	//if err != nil {
	//	//	fmt.Printf("failed to retrieve new MatchResponse - %v\n", err)
	//	//	continue
	//	//}
	//	//req.TimeReceived = time.Now()
	//	//
	//	////TODO: Set
	//	//// _, err = m.rdb.HSet(ctx, redis.MatchmakePoolUser(req.UserId), req).Result()
	//	//err = m.s.StoreKeyValue(ctx, redisutils.MatchmakePoolUser(req.UserId), req)
	//	//if err != nil {
	//	//	fmt.Printf("failed to request match - %v\n", err)
	//	//	return
	//	//}
	//	//
	//	//m.scanForMatches(ctx)
	//}
}

//func (m *Matchmaker) scanForMatches(ctx context.Context) {
//	// var cursor uint64
//
//	// keys := []string{}
//	for {
//		//TODO use identifier (skill, region, etc) to reduce the amount of requests retrieved
//
//		//TODO: RetrieveAllKeys
//		// m.s.GetAllValuesWithKeys(redis.AllMatchmakePool)
//		// res, cursor, err := m.rdb.Scan(ctx, cursor, redis.AllMatchmakePool, 100).Result() // if num keys greater than count this loops infinitely..?
//		// if err != nil {
//		// 	fmt.Printf("failed to retrieve keys - %v\n", err)
//		// 	return
//		// }
//		// keys = append(keys, res...)
//		// if cursor == 0 {
//		// 	break
//		// }
//	}
//
//	requests := []message.MatchmakingRequest{}
//	// for _, key := range keys {
//
//	// 	//TODO: GetValuesUsingKeys
//	// 	cmdReturn := m.rdb.HGetAll(ctx, key)
//	// 	var req MatchRequest
//
//	// 	if err := cmdReturn.Scan(&req); err != nil {
//	// 		fmt.Printf("failed to scan hash at key{%s} into a MatchRequest - %v\n", key, err)
//	// 		continue
//	// 	}
//	// 	if req.MatchedWith.UUID() == uuid.Nil && req.UserId.UUID() != uuid.Nil {
//	// 		requests = append(requests, req)
//	// 	}
//	// }
//	// requests = m.s.GetAllValuesWithKeys[MatchRequest](redis.AllMatchmakePool)
//	slices.SortStableFunc(requests, func(a, b message.MatchmakingRequest) int {
//		return a.TimeCreated.Compare(b.TimeCreated)
//	})
//	m.makeMatches(requests, ctx)
//}

//func (m *Matchmaker) makeMatches(requests []message.MatchmakingRequest, ctx context.Context) {
//	for i := 0; i < len(requests); i++ {
//		for j := i + 1; j < len(requests); j++ {
//			if requests[i].MatchedWith.UUID() != uuid.Nil || requests[j].MatchedWith.UUID() != uuid.Nil {
//				continue
//			}
//
//			if canMatch(requests[i], requests[j]) {
//				requests[i].MatchedWith = requests[j].UserId
//				requests[j].MatchedWith = requests[i].UserId
//
//				matchResponse := MatchResponse{
//					UserOneId:   requests[i].UserId,
//					UserOneName: requests[i].Name,
//					UserTwoId:   requests[j].UserId,
//					UserTwoName: requests[j].Name,
//				}
//				fmt.Printf("new match! - %v\n", matchResponse)
//
//				//TODO: Set
//				// _, err := m.rdb.HSet(ctx, redis.MatchmakePoolUser(requests[i].UserId), requests[i]).Result()
//				// if err != nil {
//				// 	fmt.Printf("failed to set match request - %v\n", err)
//				// 	return
//				// }
//
//				//TODO: Set
//				// _, err = m.rdb.HSet(ctx, redis.MatchmakePoolUser(requests[j].UserId), requests[j]).Result()
//				// if err != nil {
//				// 	fmt.Printf("failed to set match request - %v\n", err)
//				// 	return
//				// }
//
//				//TODO: SendMessage
//				err := m.MessageBus.Publish(ctx, redisutils.MatchFoundStream(matchResponse.UserOneId), matchResponse)
//				// _, err = m.rdb.XAdd(ctx, &goredis.XAddArgs{
//				// 	Stream: redis.MatchFoundStream(matchResponse.UserOneId),
//				// 	Values: matchResponse, //TODO add specifier to tell matchmaker what keys to pull
//				// 	ID:     "*",
//				// }).Result()
//				if err != nil {
//					fmt.Printf("error signaling to start matchmaking - %v\n", err)
//				}
//
//				//TODO: SendMessage
//				err = m.MessageBus.Publish(ctx, redisutils.MatchFoundStream(matchResponse.UserTwoId), matchResponse)
//				// _, err = m.rdb.XAdd(ctx, &goredis.XAddArgs{
//				// 	Stream: redis.MatchFoundStream(matchResponse.UserTwoId),
//				// 	Values: matchResponse, //TODO add specifier to tell matchmaker what keys to pull
//				// 	ID:     "*",
//				// }).Result()
//				if err != nil {
//					fmt.Printf("error signaling to start matchmaking - %v\n", err)
//				}
//				fmt.Println("sent match through streams")
//			}
//		}
//	}
//}

func (m *Matchmaker) processMatchmakingRequest(ctx context.Context, req message.MatchmakingRequest) {
	openRooms, err := m.RoomRepository.QueryOpenRooms(ctx, req)
	if err != nil {
		fmt.Printf("failed to retrieve open rooms - %v\n", err)
		//TODO create error channel or something in order to send an error back to the client
		return
	}

	roomFound := false
	for _, openRoom := range openRooms {
		rm, err := m.RoomRepository.JoinRoom(ctx, req, openRoom)
		if err != nil {
			if err.Error() == "ROOM_FULL" {
				continue
			}
			fmt.Printf("failed to attempt to join the open room - %v\n", err)
			continue
		}
		roomFound = true
		for _, playerId := range rm.PlayerIds {
			stream := rediskeys.MatchmakingClientMessageStream(playerId)
			if playerId == req.UserId {
				msg := NewRoomChangedMessage(rm.RoomId, rm.PlayerCount, rm.AverageSkill)
				bytes, err := json.Marshal(msg)
				if err != nil {
					fmt.Printf("failed to marshal object into json -%v\n", err)
				}
				//TODO err = m.MessageBus.Publish(ctx, stream, bytes)
				if err != nil {
					fmt.Printf("error publishing %v to %s\n", bytes, stream)
				}
				err = m.PlayerRepository.SetPlayerActive(ctx, playerId, rm.RoomId)
				if err != nil {
					fmt.Printf("error storing player in active player list - %v\n", err)
				}
			} else {
				//TODO msg := NewPlayerJoinedRoomMessage(req.UserId)
				//TODO err = m.MessageBus.Publish(ctx, stream, msg)
				//if err != nil {
				//	fmt.Printf("error publishing %v to %s\n", msg, stream)
				//}
			}
		}
		break
	}

	if !roomFound {
		//If no match is found:
		//1. A new room is created (`room:<room_id>`).
		//2. A retry timestamp is calculated.
		//3. The room ID is added to sorted set `matchmake_tasks` (scored by retry time).
		//4. `PUBLISH matchmake:notify_workers` signals workers a task is available.
		rm := room.Room{
			RoomId:            stringuuid.NewStringUUID(),
			PlayerCount:       1,
			AverageSkill:      req.Skill,
			Region:            req.Region,
			PlayerIds:         []stringuuid.StringUUID{req.UserId},
			CreatedAt:         time.Now().Unix(),
			MatchmakeAttempts: 0,
			IsFull:            0,
		}
		err = m.RoomRepository.CreateRoom(ctx, rm)
		if err != nil {
			fmt.Printf("error creating new room - %v\n", err)
		}
		msg := NewRoomChangedMessage(rm.RoomId, rm.PlayerCount, rm.AverageSkill)

		bytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("failed to marshal RoomChangedMessage into Paylaod for BaseMatchmakingClientMessage")
			panic(err) //TODO don't panic in the middle of a function..
		}

		err = m.MatchmakingClientMessageProducer.PublishTo(ctx, req.UserId, bytes)
		if err != nil {
			fmt.Printf("error publishing - %v", err)
			return
		}
		err = m.PlayerRepository.SetPlayerActive(ctx, req.UserId, rm.RoomId)
		if err != nil {
			fmt.Printf("error storing player in active player list - %v\n", err)
		}
	}
}

func (m *Matchmaker) makeMatches(ctx context.Context) error {
	// TODO:
	//1. Poll `matchmake_tasks` for the oldest task in the set
	//2. Atomically remove it from `matchmake_tasks`, insert into `matchmake_inprogress` with current time as score.
	//3. Query for compatible rooms (region/skill/time_created to adjust skill further).
	//4. If a match is found:
	//    - Perform atomic room combine (player transfer, skill/count update).
	//    - Remove affected room IDs from both queues.
	//    - Publish results to `matchmake:client_message:<client_id>`. Could be either types `room_full` or `room_combined` with an updated playercount.
	//5. If no match:
	//    - Update retry info (in room or hash).
	//    - Compute new retry time and move back to `matchmake_tasks`.
	//
	//If idle, workers block on pubsub `matchmake:notify_workers`, which is published to anytime an inactive `matchmake_inprogress` is detected or a room is created after an unsuccessful initial matchmake attempt.
	return nil
}

func (m *Matchmaker) HandleMatchmakeRequests(ctx context.Context) error {
	//m.matchmakingClientMessageConsumer.StartConsuming(ctx)
	return nil
}

func canMatch(u1 message.MatchmakingRequest, u2 message.MatchmakingRequest) bool {
	//TODO make the match rule more... something
	return u1.UserId != u2.UserId
}
