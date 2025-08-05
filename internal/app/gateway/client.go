package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	conn                         *websocket.Conn
	MatchmakingMsgCh             chan transport.WrappedConsumeMsg
	GameMsgCh                    chan transport.WrappedConsumeMsg
	ID                           stringuuid.StringUUID
	matchmakingClientMsgConsumer transport.MessageConsumer
	matchmakingServerMsgProducer transport.MessageProducer
}

var upgrader = websocket.Upgrader{}

const (
	pingInterval = 10
)

func NewClient(ctx context.Context, w http.ResponseWriter, r *http.Request, matchmakingMsgConsumerFactory transport.MessageConsumerFactory, matchmakingServerMsgProducer transport.MessageProducer) (*Client, error) {
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
	id := stringuuid.NewStringUUID() //TODO this should be stored by the client, or retrieved from a database
	matchmakingClientMsgConsumer, err := matchmakingMsgConsumerFactory.CreateConsumer(ctx, id.String())
	if err != nil {
		return c, err
	}

	c = &Client{
		conn:                         conn,
		MatchmakingMsgCh:             make(chan transport.WrappedConsumeMsg),
		GameMsgCh:                    make(chan transport.WrappedConsumeMsg),
		ID:                           id,
		matchmakingClientMsgConsumer: matchmakingClientMsgConsumer,
		matchmakingServerMsgProducer: matchmakingServerMsgProducer,
	}
	return c, nil
}

func (c *Client) PingLoop(ctx context.Context) error {
	t := time.NewTicker(time.Second * pingInterval)
	for {
		select {
		case <-t.C:
			err := c.conn.WriteMessage(websocket.PingMessage, []byte("PING"))
			if err != nil {
				return fmt.Errorf("ws{%s} failed to send PING - %v", c.ID.String(), err)
			}
			fmt.Printf("ws{%s} sent PING\n", c.ID.String())
		case <-ctx.Done():
			t.Stop()
			return nil
		}
	}
}

func (c *Client) writePump(ctx context.Context) error {
	for {
		var outMsg []byte
		var ackFunc func() error
		select {
		case <-ctx.Done():
			return nil
		case matchmakingMsg := <-c.MatchmakingMsgCh:
			ackFunc = matchmakingMsg.AckFunc
			outMsg = matchmakingMsg.Payload
		case gameMsg := <-c.GameMsgCh:
			ackFunc = gameMsg.AckFunc
			outMsg = gameMsg.Payload
		}
		err := c.conn.WriteMessage(websocket.TextMessage, outMsg)
		if err != nil {
			return fmt.Errorf("failed to write to websocket - %v", err)
		}
		err = ackFunc()
		if err != nil {
			fmt.Printf("failed to successfully call ACK for outgoing message")
		}
	}
}

func (c *Client) readPump(ctx context.Context) error {
	readChan := make(chan []byte)
	errChan := make(chan error, 1) // Buffered channel for error to prevent goroutine leak if main loop exits

	go func() {
		defer close(readChan)
		defer close(errChan)

		for {
			messageType, p, err := c.conn.ReadMessage()
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
				fmt.Printf("client %s: websocket connection closed by the client - %v\n", c.ID, err)
				return nil
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("client %s: websocket unexpectedly closed - %v\n", c.ID, err)
			} else {
				fmt.Printf("client %s: websocket encountered an error: %v\n", c.ID, err)
			}
			return err

		case bytes, ok := <-readChan:
			if !ok {
				return nil
			}
			var envelope message.Envelope
			err := json.Unmarshal(bytes, &envelope)
			if err != nil {
				fmt.Printf("client %s: failed to unmarshal message: %v\n", c.ID, err)
				continue
			}
			if err = c.RouteMessage(envelope, ctx); err != nil {
				fmt.Printf("routing message failed - %v\n", err)
			}
		}
	}
}

func (c *Client) RouteMessage(msg message.Envelope, ctx context.Context) error {
	switch msg.Type {
	case string(MatchmakingService):
		return c.matchmakingServerMsgProducer.Publish(ctx, msg.Payload)

	case string(GameService):
		//TODO err = c.m.Publish(ctx, rediskeys.GameServerMessageStream, msg.Payload)
	default:
		fmt.Println("unexpected gateway.ServiceType", msg.Type)
	}
	return nil
}

func (c *Client) listenToServices(ctx context.Context) error {
	msgCh, errCh := c.matchmakingClientMsgConsumer.StartConsuming(ctx)

	eg, gCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		for {
			select {
			case <-gCtx.Done():
				return nil
			case wrappedMatchmakingMsg, ok := <-msgCh:
				if !ok {
					return nil
				}
				wrappedMatchmakingMsg.AckFunc = func() error {
					return c.matchmakingClientMsgConsumer.AckMessage(gCtx, wrappedMatchmakingMsg.ID)
				}
				select {
				case c.MatchmakingMsgCh <- wrappedMatchmakingMsg:
				case <-gCtx.Done():
					return nil
				}
			}
		}
	})

	eg.Go(func() error {
		select {
		case <-gCtx.Done():
			return nil
		case err := <-errCh:
			return err
		}
	})

	return eg.Wait()
}
