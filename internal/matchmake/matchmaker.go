package matchmake

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/bkohler93/game-backend/internal/redis"
	"github.com/bkohler93/game-backend/pkg/interfacestruct"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Matchmaker struct {
	rdb  *goredis.Client
	mu   *sync.Mutex
	pool map[string]*MatchRequest
}

func NewMatchmaker(redisAddr string) Matchmaker {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	return Matchmaker{rdb: rdb, mu: &sync.Mutex{}, pool: make(map[string]*MatchRequest)}
}

func (m *Matchmaker) AddRequest(id string, req *MatchRequest) {
	defer m.mu.Unlock()
	m.mu.Lock()
	m.pool[id] = req
}

func (m *Matchmaker) Start(ctx context.Context) {
	for {
		fmt.Println("Listening for new matchmaking requests")
		entries, err := m.rdb.XRead(ctx, &goredis.XReadArgs{
			Streams: []string{"matchmake:request", "$"},
			Count:   1,
			Block:   0,
		}).Result()
		if err != nil {
			fmt.Printf("failed to read from matchmake:request stream - %v\n", err)
			continue
		}
		res := entries[0].Messages[0].Values
		var req MatchRequest
		err = interfacestruct.Structify(res, &req)
		if err != nil {
			fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", res, err)
			continue
		}
		req.TimeReceived = time.Now()
		_, err = m.rdb.HSet(ctx, redis.MatchmakePoolUser(req.UserId), req).Result()
		if err != nil {
			fmt.Printf("failed to request match - %v\n", err)
			return
		}

		m.scanForMatches(ctx)
	}
}

func (m *Matchmaker) scanForMatches(ctx context.Context) {
	var cursor uint64

	keys := []string{}
	for {
		//TODO use identifier (skill, region, etc) to reduce the amount of requests retrieved
		res, cursor, err := m.rdb.Scan(ctx, cursor, redis.AllMatchmakePool, 100).Result() // if num keys greater than count this loops infinitely..?
		if err != nil {
			fmt.Printf("failed to retrieve keys - %v\n", err)
			return
		}
		keys = append(keys, res...)
		if cursor == 0 {
			break
		}
	}

	requests := []MatchRequest{}
	for _, key := range keys {
		cmdReturn := m.rdb.HGetAll(ctx, key)
		var req MatchRequest

		if err := cmdReturn.Scan(&req); err != nil {
			fmt.Printf("failed to scan hash at key{%s} into a MatchRequest - %v\n", key, err)
			continue
		}
		if req.MatchedWith.UUID() == uuid.Nil && req.UserId.UUID() != uuid.Nil {
			requests = append(requests, req)
		}
	}
	slices.SortStableFunc(requests, func(a, b MatchRequest) int {
		return a.TimeReceived.Compare(b.TimeReceived)
	})
	m.makeMatches(requests, ctx)
}

func (m *Matchmaker) makeMatches(requests []MatchRequest, ctx context.Context) {
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

				_, err := m.rdb.HSet(ctx, redis.MatchmakePoolUser(requests[i].UserId), requests[i]).Result()
				if err != nil {
					fmt.Printf("failed to set match request - %v\n", err)
					return
				}

				_, err = m.rdb.HSet(ctx, redis.MatchmakePoolUser(requests[j].UserId), requests[j]).Result()
				if err != nil {
					fmt.Printf("failed to set match request - %v\n", err)
					return
				}

				_, err = m.rdb.XAdd(ctx, &goredis.XAddArgs{
					Stream: redis.MatchFoundStream(matchResponse.UserOneId),
					Values: matchResponse, //TODO add specifier to tell matchmaker what keys to pull
					ID:     "*",
				}).Result()
				if err != nil {
					fmt.Printf("error signaling to start matchmaking - %v\n", err)
				}

				_, err = m.rdb.XAdd(ctx, &goredis.XAddArgs{
					Stream: redis.MatchFoundStream(matchResponse.UserTwoId),
					Values: matchResponse, //TODO add specifier to tell matchmaker what keys to pull
					ID:     "*",
				}).Result()
				if err != nil {
					fmt.Printf("error signaling to start matchmaking - %v\n", err)
				}
				fmt.Println("sent match through streams")
			}
		}
	}
}

func canMatch(u1 MatchRequest, u2 MatchRequest) bool {
	//TODO make the match rule more... something
	return u1.UserId != u2.UserId
}
