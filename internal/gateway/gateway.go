package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/internal/matchmake"
	"github.com/bkohler93/game-backend/internal/message"
	"github.com/bkohler93/game-backend/internal/store"
	"github.com/bkohler93/game-backend/internal/utils/redisutils"
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

func NewGateway(addr string, mb message.MessageBus, s store.Store) Gateway {
	return Gateway{
		mb:   mb,
		s:    s,
		addr: addr,
		hub:  NewHub(),
	}
}

func (g *Gateway) Start(ctx context.Context) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ctx, cf := context.WithCancel(ctx)

		client, err := NewClient(w, r, g.mb)
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
		var match matchmake.MatchResponse
		err := g.mb.Consume(ctx, redisutils.MatchfoundStream, &match)
		if err != nil {
			fmt.Printf("failed to consume from matchfound stream - %v\n", err)
			continue
		}

		// entries, err := g.rdb.XRead(ctx, &goredis.XReadArgs{
		// 	Streams: []string{redis.MatchfoundStream, "$"},
		// 	Count:   1,
		// 	Block:   0,
		// }).Result()
		// if err != nil {
		// 	fmt.Printf("failed to read from matchmake:found stream - %v\n", err)
		// 	continue
		// }
		// res := entries[0].Messages[0].Values

		// var match matchmake.MatchResponse
		// err = interfacestruct.Structify(res, &match)
		// if err != nil {
		// 	fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", res, err)
		// 	continue
		// }
		if _, ok := g.hub.Clients[match.UserOneId]; ok {
			g.hub.Clients[match.UserOneId].MatchmakingMsgCh <- match
		}

		if _, ok := g.hub.Clients[match.UserTwoId]; ok {
			g.hub.Clients[match.UserTwoId].MatchmakingMsgCh <- match
		}
	}
}
