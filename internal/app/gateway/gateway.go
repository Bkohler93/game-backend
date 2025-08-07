package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bkohler93/game-backend/internal/app/gateway/client"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type Gateway struct {
	clientTransportBusFactory *client.ClientTransportBusFactory
	roomRepository            *room.Repository
	addr                      string
	hub                       *Hub
}

func NewGateway(addr string, rr *room.Repository, clientTransportBusFactory *client.ClientTransportBusFactory) Gateway {
	return Gateway{
		clientTransportBusFactory: clientTransportBusFactory,
		roomRepository:            rr,
		addr:                      addr,
		hub:                       NewHub(),
	}
}

func (g *Gateway) Start(ctx context.Context) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		eg, ctx := errgroup.WithContext(ctx)

		c, err := client.NewClient(ctx, w, r, g.clientTransportBusFactory)
		if err != nil {
			fmt.Printf("failed to initialize client websocket - %v\n", err)
			return
		}
		defer func(conn *websocket.Conn) {
			err := conn.Close()
			if err != nil {
				fmt.Printf("failed to close connection - %v\n", err)
			}
		}(c.Conn)

		g.hub.RegisterCh <- c
		eg.Go(func() error {
			return c.PingLoop(ctx)
		})

		eg.Go(func() error {
			return c.WritePump(ctx)
		})

		eg.Go(func() error {
			return c.ReadPump(ctx)
		})

		eg.Go(func() error {
			return c.ListenToServices(ctx)
		})

		if err = eg.Wait(); err != nil {
			fmt.Printf("worker encountered an error - %v", err)
		} else {
			fmt.Println("workers ended with no errors")
		}

		g.hub.UnregisterCh <- c
	})

	fmt.Printf("listening on %s for new matchmaking requests from clients\n", g.addr)
	err := http.ListenAndServe("0.0.0.0:"+g.addr, nil)
	if err != nil {
		log.Fatalf("error creating server - %v", err)
	}
}
