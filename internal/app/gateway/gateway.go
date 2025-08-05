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
	"golang.org/x/sync/errgroup"
)

const (
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

type Gateway struct {
	matchmakingClientMessageConsumerFactory transport.MessageConsumerFactory
	matchmakingServerMessageProducer        transport.MessageProducer
	roomRepository                          *room.Repository
	addr                                    string
	hub                                     *Hub
}

func NewGateway(addr string, rr *room.Repository, matchmakingClientMessageConsumerFactory transport.MessageConsumerFactory, matchmakingServerMessageProducer transport.MessageProducer) Gateway {
	return Gateway{
		roomRepository:                          rr,
		addr:                                    addr,
		matchmakingClientMessageConsumerFactory: matchmakingClientMessageConsumerFactory,
		matchmakingServerMessageProducer:        matchmakingServerMessageProducer,
		hub:                                     NewHub(),
	}
}

func (g *Gateway) Start(ctx context.Context) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		eg, ctx := errgroup.WithContext(ctx)

		client, err := NewClient(ctx, w, r, g.matchmakingClientMessageConsumerFactory, g.matchmakingServerMessageProducer)
		if err != nil {
			fmt.Printf("failed to initialize client websocket - %v\n", err)
			return
		}
		defer func(conn *websocket.Conn) {
			err := conn.Close()
			if err != nil {
				fmt.Printf("failed to close connection - %v\n", err)
			}
		}(client.conn)

		g.hub.RegisterCh <- client
		eg.Go(func() error {
			return client.PingLoop(ctx)
		})

		eg.Go(func() error {
			return client.writePump(ctx)
		})

		eg.Go(func() error {
			return client.readPump(ctx)
		})

		eg.Go(func() error {
			return client.listenToServices(ctx)
		})

		if err = eg.Wait(); err != nil {
			fmt.Printf("worker encountered an error - %v", err)
		} else {
			fmt.Println("workers ended with no errors")
		}

		g.hub.UnregisterCh <- client
	})

	fmt.Printf("listening on %s for new matchmaking requests from clients\n", g.addr)
	err := http.ListenAndServe("0.0.0.0:"+g.addr, nil)
	if err != nil {
		log.Fatalf("error creating server - %v", err)
	}
}
