package gateway

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type Gateway struct {
	//clientTransportBusFactory *client.ClientTransportBusFactory
	roomRepository *room.Repository
	addr           string
	hub            *Hub
}

func NewGateway(addr string, rr *room.Repository, transportFactory *TransportFactory) Gateway {
	r := &Router{
		transportFactory: transportFactory,
	}
	return Gateway{
		//clientTransportBusFactory: clientTransportBusFactory,
		roomRepository: rr,
		addr:           addr,
		hub:            NewHub(r),
	}
}

func (g *Gateway) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		log.Println("health check received at \"\\\"")
		_, err := fmt.Fprintln(w, "OK")
		if err != nil {
			log.Println(err)
		}
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		eg, ctx := errgroup.WithContext(ctx)

		c, err := NewClient(ctx, w, r)
		if err != nil {
			log.Printf("failed to initialize client websocket - %v\n", err)
			return
		}
		defer func(conn *websocket.Conn) {
			err := conn.Close()
			if err != nil {
				log.Printf("failed to close connection - %v\n", err)
			}
		}(c.Conn)

		g.hub.RegisterCh <- c
		eg.Go(func() error {
			return c.PingLoop(ctx)
		})

		eg.Go(func() error {
			return c.WritePump(ctx)
		})

		eg.Go(func() error { return c.ReadPump(ctx) })

		//eg.Go(func() error {
		//	return c.ListenToServices(ctx)
		//})

		if err = eg.Wait(); err != nil {
			if errors.Is(err, ErrClientClosedConnection) {
			} else {
				log.Println("worker ended due to unknown error -", err)
			}
		}

		g.hub.UnregisterCh <- c
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", g.addr),
		Handler: mux,
	}

	go func() {
		log.Println("Starting server on :" + g.addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped")
}
