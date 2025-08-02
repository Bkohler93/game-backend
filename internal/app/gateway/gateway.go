package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/gorilla/websocket"
)

const (
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

type Gateway struct {
	//mb             message.MessageBus
	matchmakingMessageConsumerFactory transport.MatchmakingClientMessageConsumerFactory
	roomRepository                    *room.Repository
	addr                              string
	hub                               *Hub
}

func NewGateway(addr string, rr *room.Repository, matchmakingMessageConsumerFactory transport.MatchmakingClientMessageConsumerFactory) Gateway {
	return Gateway{
		roomRepository:                    rr,
		addr:                              addr,
		matchmakingMessageConsumerFactory: matchmakingMessageConsumerFactory,
		hub:                               NewHub(),
	}
}

func (g *Gateway) Start(ctx context.Context) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ctx, cf := context.WithCancel(ctx)

		client, err := NewClient(ctx, w, r, g.matchmakingMessageConsumerFactory)
		if err != nil {
			fmt.Printf("failed to initialize client websocket - %v\n", err)
			cf()
			return
		}
		defer func(conn *websocket.Conn) {
			err := conn.Close()
			if err != nil {
				fmt.Printf("failed to close connection - %v\n", err)
			}
		}(client.conn)

		g.hub.RegisterCh <- client

		go client.PingLoop(ctx, cf)

		go client.writePump(ctx, cf)
		go client.readPump(ctx, cf)

		go client.listenToRedis(ctx, cf)

		<-ctx.Done()

		g.hub.UnregisterCh <- client
	})

	fmt.Printf("listening on %s for new matchmaking requests from clients\n", g.addr)
	err := http.ListenAndServe("0.0.0.0:"+g.addr, nil)
	if err != nil {
		log.Fatalf("error creating server - %v", err)
	}
}
