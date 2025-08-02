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
)

type Client struct {
	conn                   *websocket.Conn
	MatchmakingMsgCh       chan message.BaseMatchmakingClientMessage
	GameMsgCh              chan message.BaseGameClientMessage
	ID                     stringuuid.StringUUID
	matchmakingMsgConsumer transport.MatchmakingClientMessageConsumer
	// rdb              *redis.RedisClient
}

var upgrader = websocket.Upgrader{}

const (
	pingInterval = 10
)

func NewClient(ctx context.Context, w http.ResponseWriter, r *http.Request, matchmakingMsgFactory transport.MatchmakingClientMessageConsumerFactory) (*Client, error) {
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
	matchmakingMsgConsumer, err := matchmakingMsgFactory.CreateConsumer(ctx, id)
	if err != nil {
		return c, err
	}

	c = &Client{
		conn:                   conn,
		MatchmakingMsgCh:       make(chan message.BaseMatchmakingClientMessage),
		GameMsgCh:              make(chan message.BaseGameClientMessage),
		ID:                     id,
		matchmakingMsgConsumer: matchmakingMsgConsumer,
	}
	return c, nil
}

func (c *Client) PingLoop(ctx context.Context, cf context.CancelFunc) {
	t := time.NewTicker(time.Second * pingInterval)
	fmt.Printf("client{%s} spawned PingLoop\n", c.ID.String())
	for {
		select {
		case <-t.C:
			err := c.conn.WriteMessage(websocket.PingMessage, []byte("PING"))
			if err != nil {
				fmt.Printf("client{%s} failed to send PING\n", c.ID.String())
				cf()
			}
			fmt.Printf("client{%s} sent PING\n", c.ID.String())
		case <-ctx.Done():
			t.Stop()
			return
		}
	}
}

func (c *Client) writePump(ctx context.Context, connCancelFunc context.CancelFunc) {
	var bytes []byte
	var err error
	for {
		var message BaseMessage
		select {
		case <-ctx.Done():
			return
		case msg := <-c.MatchmakingMsgCh:
			bytes, err = json.Marshal(msg)
			if err != nil {
				fmt.Printf("failed to marshal outgoing matchmaking msg - %v\n", err)
				continue
			}
			message.Payload = bytes
		case msg := <-c.GameMsgCh:
			bytes, err = json.Marshal(msg)
			if err != nil {
				fmt.Printf("failed to marshal outgoing game message msg - %v\n", err)
				continue
			}
			message.Payload = bytes
		}
		finalBytes, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("failed to marshal final output message - %v\n", err)
			continue
		}
		fmt.Printf("sending %s\n", string(finalBytes))
		err = c.conn.WriteMessage(websocket.TextMessage, finalBytes)
		if err != nil {
			fmt.Println("failed to write to websocket", err)
			connCancelFunc()
			return
		}
	}
}

func (c *Client) readPump(ctx context.Context, connCancelFunc context.CancelFunc) {
	for {
		readChan := make(chan []byte)
		errChan := make(chan error, 1) // Buffered channel for error to prevent goroutine leak if main loop exits

		go func() {
			messageType, p, err := c.conn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				readChan <- p
			} else {
				errChan <- fmt.Errorf("unhandled websocket message type: %d", messageType)
			}
		}()

		select {
		case <-ctx.Done():
			return
		case bytes := <-readChan:
			var bm BaseMessage
			err := json.Unmarshal(bytes, &bm)
			if err != nil {
				fmt.Printf("client %s: failed to unmarshal message: %v\n", c.ID, err)
				connCancelFunc()
				return
			}
			c.ForwardMessageToService(bm, ctx, connCancelFunc)
		case err := <-errChan:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				fmt.Printf("client %s: websocket connection closed by the client - %v\n", c.ID, err.Error())
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("client %s: websocket unexpectedly closed - %v\n", c.ID, err.Error())
			} else {
				fmt.Printf("client %s: websocket encountered an error: %v\n", c.ID, err)
			}
			connCancelFunc()
			return
		}
	}
}

func (c *Client) ForwardMessageToService(msg BaseMessage, ctx context.Context, connCancelFunc context.CancelFunc) {
	switch msg.ServiceType {
	case MatchmakingService:
		var action message.GameClientActionBase

		err := json.Unmarshal(msg.Payload, &action)
		if err != nil {
			fmt.Printf("error unmarsheling gameplay message %v\n", err)
			connCancelFunc()
		}
		//TODO err = c.m.Publish(ctx, rediskeys.GameServerMessageStream(action.GameID), action)
		if err != nil {
			fmt.Printf("error forwarding gameplay message %v\n", err)
			connCancelFunc()
		}
		// _, err := c.rdb.XAdd(ctx, &goredis.XAddArgs{
		// 	Stream: redis.GameServerMessageStream(action.GameID),
		// 	Values: action,
		// 	ID:     "*",
		// }).Result()
	case GameService:
		var serverMsg message.BaseMatchmakingServerMessage
		err := json.Unmarshal(msg.Payload, &serverMsg)
		if err != nil {
			fmt.Printf("failed to unmarshal msg payload - %v", err)
			connCancelFunc()
			return
		}
		//TODO err = c.m.Publish(ctx, rediskeys.MatchmakingServerMessageStream, serverMsg)
	default:
		fmt.Println("unexpected gateway.ServiceType", msg.ServiceType)
		connCancelFunc()
	}
}

func (c *Client) listenToRedis(ctx context.Context, connCancelFunc context.CancelFunc) {
	// read from matchmake client message stream
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				//err := c.m.Consume(ctx, rediskeys.MatchmakingClientMessageStream(c.ID), &msg)
				msg, err := c.matchmakingMsgConsumer.Consume(ctx)
				// entries, err := c.rdb.XRead(ctx, &goredis.XReadArgs{
				// 	Streams: []string{redis.MatchFoundStream(c.ID), "$"},
				// 	Count:   1,
				// 	Block:   0,
				// }).Result()
				// if err != nil {
				// 	fmt.Printf("failed to read from match found stream - %v\n", err)
				// 	continue
				// }
				// if len(entries) == 0 || len(entries[0].Messages) == 0 {
				// 	fmt.Printf("read from match found stream zero results")
				// 	continue
				// }
				// data := entries[0].Messages[0].Values

				// res := matchmake.EmptyMatchMadeMessage()
				// err := interfacestruct.Structify(data, &res)
				if err != nil {
					fmt.Printf("failed to get new MatchResponse - %v\n", err)
				}
				c.MatchmakingMsgCh <- msg
				return
			}
		}
	}()

	// read from game client message stream
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				//TODO: ReadFromBlocking
				//res := game.ServerResponse{}
				//err := c.m.Consume(ctx, rediskeys.GameClientMessageStream(c.ID), &res)
				// entries, err := c.rdb.XRead(ctx, &goredis.XReadArgs{
				// 	Streams: []string{redis.GameServerResponseStream(c.ID), "$"},
				// 	Count:   1,
				// 	Block:   0,
				// }).Result()
				// if err != nil {
				// 	fmt.Printf("failed to read from game server response stream - %v\n", err)
				// }
				// data := entries[0].Messages[0].Values
				// if len(entries) == 0 || len(entries[0].Messages) == 0 {
				// 	fmt.Printf("read from game server response stream with zero results")
				// 	continue
				// }

				// var res game.ServerResponse
				// err = interfacestruct.Structify(data, &res)
				//if err != nil {
				//	fmt.Printf("failed to new MatchResponse - %v\n", err)
				//}
				//c.GameMsgCh <- res
			}
		}
	}()
}
