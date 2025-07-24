package gateway

import (
	"fmt"
	"sync"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type Hub struct {
	Clients map[stringuuid.StringUUID]*Client
	mu      sync.Mutex

	RegisterCh   chan *Client
	UnregisterCh chan *Client
}

func NewHub() *Hub {
	h := &Hub{
		Clients:      map[stringuuid.StringUUID]*Client{},
		RegisterCh:   make(chan *Client),
		UnregisterCh: make(chan *Client),
	}

	go func() {
		for {
			select {
			case c := <-h.RegisterCh:
				h.Clients[c.ID] = c
			case c := <-h.UnregisterCh:
				err := c.conn.Close()
				if err != nil {
					fmt.Printf("error closing client's(%s) websocket conn - %v", c.ID.String(), err)
				}
				delete(h.Clients, c.ID)
			}
		}
	}()

	return h
}
