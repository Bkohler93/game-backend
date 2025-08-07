package gateway

import (
	"fmt"

	"github.com/bkohler93/game-backend/internal/app/gateway/client"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type Hub struct {
	Clients map[uuidstring.ID]*client.Client

	RegisterCh   chan *client.Client
	UnregisterCh chan *client.Client
}

func NewHub() *Hub {
	h := &Hub{
		Clients:      map[uuidstring.ID]*client.Client{},
		RegisterCh:   make(chan *client.Client),
		UnregisterCh: make(chan *client.Client),
	}

	go func() {
		for {
			select {
			case c := <-h.RegisterCh:
				h.Clients[c.ID] = c
			case c := <-h.UnregisterCh:
				err := c.Conn.Close()
				if err != nil {
					fmt.Printf("error closing client's(%s) websocket conn - %v", c.ID.String(), err)
				}
				delete(h.Clients, c.ID)
			}
		}
	}()

	return h
}
