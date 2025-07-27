package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/internal/matchmake"
	"github.com/bkohler93/game-backend/internal/message"
	"github.com/bkohler93/game-backend/internal/redis"
	"github.com/bkohler93/game-backend/pkg/interfacestruct"
	"github.com/fatihkahveci/simple-matchmaking/store"
)

const (
	pongWait = 60 * time.Second

	maxMessageSize = 512
)

type Gateway struct {
	mb   message.MessageBus
	s    store.Store
	addr string
	hub  *Hub
}

func NewGateway(addr, redisClient *redis.RedisClient) Gateway {
	return Gateway{
		rdb: redisClient,
		hub: NewHub(),
	}
}

func (g *Gateway) Start(ctx context.Context) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ctx, cf := context.WithCancel(ctx)

		client, err := NewClient(w, r, g.rdb)
		if err != nil {
			fmt.Printf("failed to initialize client websocket - %v\n", err)
			cf()
			return
		}
		defer client.conn.Close()

		g.hub.RegisterCh <- client

		go client.PingLoop(ctx, cf)

		go client.writePump(ctx, cf)
		go client.readPump(ctx, cf)

		go client.listenToRedis(ctx, cf)

		<-ctx.Done()

		g.hub.UnregisterCh <- client
	})

	go g.listenForMatches(ctx)

	fmt.Printf("listening on %s for new matchmaking requests from clients\n", g.addr)
	err := http.ListenAndServe("0.0.0.0:"+g.addr, nil)
	if err != nil {
		log.Fatalf("error creating server - %v", err)
	}
}

func (g *Gateway) listenForMatches(ctx context.Context) {
	for {
		fmt.Println("waiting for match to come through stream")

		//TODO: ReadFromStreamBlocking()
		entries, err := g.rdb.XRead(ctx, &goredis.XReadArgs{
			Streams: []string{redis.MatchfoundStream, "$"},
			Count:   1,
			Block:   0,
		}).Result()
		if err != nil {
			fmt.Printf("failed to read from matchmake:found stream - %v\n", err)
			continue
		}
		res := entries[0].Messages[0].Values

		var match matchmake.MatchResponse
		err = interfacestruct.Structify(res, &match)
		if err != nil {
			fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", res, err)
			continue
		}
		g.hub.Clients[match.UserOneId].MatchmakingMsgCh <- match
		g.hub.Clients[match.UserTwoId].MatchmakingMsgCh <- match
	}
}
