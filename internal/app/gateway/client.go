package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var (
	ErrClientClosedConnection = errors.New("client disconnected")
	ErrClientDisconnected     = errors.New("client disconnected due to abnormal closure")
)

type Client struct {
	Conn      *websocket.Conn
	writeChan chan *message.EnvelopeContext
	ID        uuidstring.ID
	RoomID    uuidstring.ID
	routeChan chan *message.EnvelopeContext
	errChan   chan error
}

var upgrader = websocket.Upgrader{}

const (
	pingInterval   = 10
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

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

	userId, err := receiveAndVerifyJWT(conn)
	if err != nil {
		return c, err
	}

	c = &Client{
		Conn:      conn,
		writeChan: make(chan *message.EnvelopeContext),
		routeChan: make(chan *message.EnvelopeContext),
		errChan:   make(chan error),
		ID:        userId,
	}
	return c, nil
}

func receiveAndVerifyJWT(conn *websocket.Conn) (uuidstring.ID, error) {
	var id uuidstring.ID
	messageType, p, err := conn.ReadMessage()
	if err != nil {
		return id, err
	}

	if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
		return id, fmt.Errorf("unhandled message type: %d", messageType)
	}

	var connectingMessage *ConnectingMessage
	err = json.Unmarshal(p, &connectingMessage)
	if err != nil {
		return id, err
	}
	token, err := jwt.Parse(connectingMessage.JwtString, func(token *jwt.Token) (any, error) {
		if os.Getenv("JWT_SECRET") == "" {
			return "", fmt.Errorf("JWT_SECRET is not set")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return id, err
	}

	switch {
	case token.Valid:
		idStr, err := token.Claims.GetSubject()
		if err != nil {
			return uuidstring.NewID(), err
		}
		msg := NewAuthenticatedMessage(uuidstring.ID(idStr))
		bytes, err := json.Marshal(msg)
		if err != nil {
			return uuidstring.NewID(), err
		}
		id = msg.UserId

		err = conn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			log.Printf("error sending logged in message - %v\n", err)
		}

		return id, nil
	case errors.Is(err, jwt.ErrTokenMalformed):
		return id, err
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return id, err
	case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
		return id, err
	default:
		return id, fmt.Errorf("could not handle token")
	}
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
		select {
		case <-ctx.Done():
			return nil
		case outMsg := <-c.writeChan:
			err := c.Conn.WriteMessage(websocket.TextMessage, outMsg.Env.Payload)
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
	errChan := make(chan error, 1)

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
				c.errChan <- ErrClientClosedConnection
				return ErrClientClosedConnection
			}
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.errChan <- ErrClientDisconnected
				return ErrClientDisconnected
			} else {
				log.Printf("client[%s]: websocket encountered an unknown error - %v\n", c.ID, err)
			}
			return err

		case bytes, ok := <-readChan:
			if !ok {
				return nil
			}
			routingDetails := struct {
				Type string `json:"$type"`
			}{}
			err := json.Unmarshal(bytes, &routingDetails)
			if err != nil {
				log.Printf("client %s: failed to unmarshal message: %v\n", c.ID, err)
				continue
			}
			envelope := &message.Envelope{
				Type:     getServiceType(routingDetails.Type),
				Payload:  bytes,
				MetaData: nil,
			}

			msgContext := message.NewNoAckEnvelopeContext(envelope)
			c.routeChan <- msgContext
		}
	}
}
