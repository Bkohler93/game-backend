package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

var (
	ErrClientClosedConnection = errors.New("client disconnected")
)

type Client struct {
	Conn *websocket.Conn
	//MatchmakingMsgCh chan transport.AckableMessage
	//GameMsgCh        chan transport.AckableMessage
	outChan   chan transport.AckableMessage
	ID        uuidstring.ID
	routeChan chan message.Envelope

	//TransportBus     *TransportBus
}

var upgrader = websocket.Upgrader{}

const (
	pingInterval   = 10
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

// func NewClient(ctx context.Context, w http.ResponseWriter, r *http.Request, transportBusFactory *ClientTransportBusFactory) (*Client, error) {
func NewClient(ctx context.Context, w http.ResponseWriter, r *http.Request) (*Client, error) {
	var c *Client
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return c, err
	}

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	id := uuidstring.NewID() //TODO this should be stored by the client, or retrieved from a database
	//transportBus := transportBusFactory.NewClientTransportBus(id)

	//TODO this is temporary until we figure out where the id comes from
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("id=%s", id)))

	c = &Client{
		Conn: conn,
		//MatchmakingMsgCh: make(chan transport.AckableMessage),
		//GameMsgCh:        make(chan transport.AckableMessage),
		outChan: make(chan transport.AckableMessage),
		ID:      id,
		//TransportBus:     transportBus,
		//matchmakingClientMsgConsumer: matchmakingClientMsgConsumer,
		//matchmakingServerMsgProducer: matchmakingServerMsgProducer,
	}
	return c, nil
}

func (c *Client) PingLoop(ctx context.Context) error {
	t := time.NewTicker(time.Second * pingInterval)
	for {
		select {
		case <-t.C:
			err := c.Conn.WriteMessage(websocket.PingMessage, []byte("PING"))
			if err != nil {
				if errors.Is(err, websocket.ErrCloseSent) {
					return ErrClientClosedConnection
				}
				return fmt.Errorf("ws{%s} failed to send PING - %v", c.ID.String(), err)
			}
		case <-ctx.Done():
			t.Stop()
			return nil
		}
	}
}

func (c *Client) WritePump(ctx context.Context) error {

	for {
		//var outMsg []byte
		//var ackFunc func() error
		select {
		case <-ctx.Done():
			return nil
		case outMsg := <-c.outChan:
			bytes := outMsg.Payload.([]byte)
			//case matchmakingMsg := <-c.MatchmakingMsgCh:
			//	messageId := matchmakingMsg.ID
			//	outMsg = matchmakingMsg.Payload.([]byte)
			//	ackFunc = func() error {
			//		err := c.TransportBus.AckMatchmakingMsg(ctx, messageId)
			//		return err
			//	}
			//case gameMsg := <-c.GameMsgCh:
			//	outMsg = gameMsg.Payload.([]byte)
			//	ackFunc = func() error {
			//		//TODO return c.TransportBus.AckGameMsg(ctx, messageId)
			//		return nil
			//	}
			//}
			err := c.Conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				return fmt.Errorf("failed to write to websocket - %v", err)
			}
			err = outMsg.AckFunc(ctx)
			if err != nil {
				log.Println("failed to successfully call ACK for outgoing message")
			}
		}
	}
}

func (c *Client) ReadPump(ctx context.Context) error {
	readChan := make(chan []byte)
	errChan := make(chan error, 1) // Buffered channel for error to prevent goroutine leak if main loop exits

	go func() {
		defer close(readChan)
		defer close(errChan)

		for {
			messageType, p, err := c.Conn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}

			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				errChan <- fmt.Errorf("unhandled message type: %d", messageType)
				return
			}
			readChan <- p
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case err := <-errChan:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return ErrClientClosedConnection
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("client[%s]: websocket unexpectedly closed - %v\n", c.ID, err)
			} else {
				log.Printf("client[%s]: websocket encountered an error: %v\n", c.ID, err)
			}
			return err

		case bytes, ok := <-readChan:
			if !ok {
				return nil
			}
			var envelope message.Envelope
			err := json.Unmarshal(bytes, &envelope)
			if err != nil {
				log.Printf("client %s: failed to unmarshal message: %v\n", c.ID, err)
				continue
			}
			c.routeChan <- envelope
			//if err = c.RouteMessage(envelope, ctx); err != nil {
			//	log.Printf("routing message failed - %v\n", err)
			//}
		}
	}
}

func (c *Client) RouteMessage(msg message.Envelope, ctx context.Context) error {
	switch msg.Type {
	case string(message.MatchmakingService):
		return c.TransportBus.SendMatchmakingServerMessage(ctx, msg.Payload)

	case string(message.GameService):
		//TODO err = c.m.Send(ctx, rediskeys.GameServerMessageStream, msg.Payload)
	default:
		log.Println("unexpected gateway.ServiceType", msg.Type)
	}
	return nil
}

func (c *Client) ListenToServices(ctx context.Context) error {
	matchmakingMsgCh, matchmakingErrCh := c.TransportBus.StartReceivingMatchmakingClientMessages(ctx)

	eg, gCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		for {
			select {
			case <-gCtx.Done():
				return nil
			case wrappedMatchmakingMsg, open := <-matchmakingMsgCh:
				if !open {
					return nil
				}
				c.MatchmakingMsgCh <- wrappedMatchmakingMsg
			}
		}
	})

	eg.Go(func() error {
		select {
		case <-gCtx.Done():
			return nil
		case err := <-matchmakingErrCh:
			return err
		}
	})

	return eg.Wait()
}
