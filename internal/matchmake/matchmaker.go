package matchmake

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/bkohler93/game-backend/internal/message"
	"github.com/bkohler93/game-backend/internal/room"
	"github.com/bkohler93/game-backend/internal/utils/redisutils"
	"github.com/google/uuid"
)

type Matchmaker struct {
	mb             message.MessageBus
	roomRepository *room.Repository
}

func NewMatchmaker(mb message.MessageBus, rr *room.Repository) *Matchmaker {

	return &Matchmaker{mb: mb, roomRepository: rr}
}

func (m *Matchmaker) Start(ctx context.Context) {
	for {
		fmt.Println("Listening for new Matchmaking messages.")

		var msg message.BaseMatchmakingServerMessage
		err := m.mb.Consume(ctx, redisutils.MatchmakingServerMessageStream(), &msg)
		switch msg.Type {
		case message.ServerMessageTypeMatchmakingRequest:
			var req message.MatchmakingRequest
			err := json.Unmarshal(msg.Payload, &req)
			if err != nil {
				fmt.Printf("Error unmarshalling matchmaking request: %v\n", err)
				continue
			}
			go m.processMatchmakingRequest(req)
			break
		case message.ServerMessageTypeMatchmakingExit:
		}
		// entries, err := m.rdb.XRead(ctx, &goredis.XReadArgs{
		// 	Streams: []string{"matchmake:request", "$"},
		// 	Count:   1,
		// 	Block:   0,
		// }).Result()
		// if err != nil {
		// 	fmt.Printf("failed to read from matchmake:request stream - %v\n", err)
		// 	continue
		// }
		// res := entries[0].Messages[0].Values
		// var req MatchRequest
		// err = interfacestruct.Structify(res, &req)
		if err != nil {
			fmt.Printf("failed to retrieve new MatchResponse - %v\n", err)
			continue
		}
		req.TimeReceived = time.Now()

		//TODO: Set
		// _, err = m.rdb.HSet(ctx, redis.MatchmakePoolUser(req.UserId), req).Result()
		err = m.s.StoreKeyValue(ctx, redisutils.MatchmakePoolUser(req.UserId), req)
		if err != nil {
			fmt.Printf("failed to request match - %v\n", err)
			return
		}

		m.scanForMatches(ctx)
	}
}

func (m *Matchmaker) scanForMatches(ctx context.Context) {
	// var cursor uint64

	// keys := []string{}
	for {
		//TODO use identifier (skill, region, etc) to reduce the amount of requests retrieved

		//TODO: RetrieveAllKeys
		// m.s.GetAllValuesWithKeys(redis.AllMatchmakePool)
		// res, cursor, err := m.rdb.Scan(ctx, cursor, redis.AllMatchmakePool, 100).Result() // if num keys greater than count this loops infinitely..?
		// if err != nil {
		// 	fmt.Printf("failed to retrieve keys - %v\n", err)
		// 	return
		// }
		// keys = append(keys, res...)
		// if cursor == 0 {
		// 	break
		// }
	}

	requests := []message.MatchmakingRequest{}
	// for _, key := range keys {

	// 	//TODO: GetValuesUsingKeys
	// 	cmdReturn := m.rdb.HGetAll(ctx, key)
	// 	var req MatchRequest

	// 	if err := cmdReturn.Scan(&req); err != nil {
	// 		fmt.Printf("failed to scan hash at key{%s} into a MatchRequest - %v\n", key, err)
	// 		continue
	// 	}
	// 	if req.MatchedWith.UUID() == uuid.Nil && req.UserId.UUID() != uuid.Nil {
	// 		requests = append(requests, req)
	// 	}
	// }
	// requests = m.s.GetAllValuesWithKeys[MatchRequest](redis.AllMatchmakePool)
	slices.SortStableFunc(requests, func(a, b message.MatchmakingRequest) int {
		return a.TimeCreated.Compare(b.TimeCreated)
	})
	m.makeMatches(requests, ctx)
}

func (m *Matchmaker) makeMatches(requests []message.MatchmakingRequest, ctx context.Context) {
	for i := 0; i < len(requests); i++ {
		for j := i + 1; j < len(requests); j++ {
			if requests[i].MatchedWith.UUID() != uuid.Nil || requests[j].MatchedWith.UUID() != uuid.Nil {
				continue
			}

			if canMatch(requests[i], requests[j]) {
				requests[i].MatchedWith = requests[j].UserId
				requests[j].MatchedWith = requests[i].UserId

				matchResponse := MatchResponse{
					UserOneId:   requests[i].UserId,
					UserOneName: requests[i].Name,
					UserTwoId:   requests[j].UserId,
					UserTwoName: requests[j].Name,
				}
				fmt.Printf("new match! - %v\n", matchResponse)

				//TODO: Set
				// _, err := m.rdb.HSet(ctx, redis.MatchmakePoolUser(requests[i].UserId), requests[i]).Result()
				// if err != nil {
				// 	fmt.Printf("failed to set match request - %v\n", err)
				// 	return
				// }

				//TODO: Set
				// _, err = m.rdb.HSet(ctx, redis.MatchmakePoolUser(requests[j].UserId), requests[j]).Result()
				// if err != nil {
				// 	fmt.Printf("failed to set match request - %v\n", err)
				// 	return
				// }

				//TODO: SendMessage
				err := m.mb.Publish(ctx, redisutils.MatchFoundStream(matchResponse.UserOneId), matchResponse)
				// _, err = m.rdb.XAdd(ctx, &goredis.XAddArgs{
				// 	Stream: redis.MatchFoundStream(matchResponse.UserOneId),
				// 	Values: matchResponse, //TODO add specifier to tell matchmaker what keys to pull
				// 	ID:     "*",
				// }).Result()
				if err != nil {
					fmt.Printf("error signaling to start matchmaking - %v\n", err)
				}

				//TODO: SendMessage
				err = m.mb.Publish(ctx, redisutils.MatchFoundStream(matchResponse.UserTwoId), matchResponse)
				// _, err = m.rdb.XAdd(ctx, &goredis.XAddArgs{
				// 	Stream: redis.MatchFoundStream(matchResponse.UserTwoId),
				// 	Values: matchResponse, //TODO add specifier to tell matchmaker what keys to pull
				// 	ID:     "*",
				// }).Result()
				if err != nil {
					fmt.Printf("error signaling to start matchmaking - %v\n", err)
				}
				fmt.Println("sent match through streams")
			}
		}
	}
}

func (m *Matchmaker) processMatchmakingRequest(ctx context.Context, req message.MatchmakingRequest) {
	openRooms, err := m.roomRepository.QueryOpenRooms(ctx, req)
	if err != nil {
		fmt.Printf("failed to retrieve open rooms - %v\n", err)
		//TODO create error channel or something in order to send an error back to the client
		return
	}
	for _, openRoom := range openRooms {
		room, err := m.roomRepository.TryJoinRoom(req, openRoom)
		if err != nil  && err.String() != "room full" {
			fmt.Printf("failed to attempt to join the open room - %v\n", err)
			continue
		} 
		if room == nil {
			fmt.Printf("unable to join room - %v", err.String())
		}	
	}
}

func canMatch(u1 message.MatchmakingRequest, u2 message.MatchmakingRequest) bool {
	//TODO make the match rule more... something
	return u1.UserId != u2.UserId
}
