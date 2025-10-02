package gateway

import (
	"context"
	"errors"
	"log"
	"net"
)

type Hub struct {
	router  *Router
	Clients map[*Client]context.CancelFunc

	RegisterCh   chan *Client
	UnregisterCh chan *Client
}

func NewHub(r *Router) *Hub {
	h := &Hub{
		Clients:      make(map[*Client]context.CancelFunc),
		RegisterCh:   make(chan *Client),
		UnregisterCh: make(chan *Client),
		router:       r,
	}

	go func() {
		for {
			select {
			case c := <-h.RegisterCh:
				ctx, cancelFunc := context.WithCancel(context.Background())
				h.Clients[c] = cancelFunc

				go h.router.RouteClientTraffic(ctx, c)

				log.Printf("registered client[%s]\n", c.ID)
			case c := <-h.UnregisterCh:
				err := c.Conn.Close()
				if err != nil {
					if !errors.Is(err, net.ErrClosed) {
						log.Printf("error closing client's(%s) websocket conn - %v", c.ID.String(), err)
					}
				}
				cancelFunc := h.Clients[c]
				cancelFunc()
				delete(h.Clients, c)
				log.Printf("unregistered client[%s]\n", c.ID)
			}
		}
	}()

	return h
}
