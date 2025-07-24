package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bkohler93/game-backend/internal/game"
	"github.com/bkohler93/game-backend/internal/matchmake"
	"github.com/bkohler93/game-backend/internal/redis"
	"github.com/bkohler93/game-backend/pkg/interfacestruct"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/gorilla/websocket"
	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	conn             *websocket.Conn
	MatchmakingMsgCh chan (matchmake.MatchResponse)
	GameMsgCh        chan (game.ServerResponse)
	ID               stringuuid.StringUUID
	rdb              *goredis.Client
}

func NewClient(c *websocket.Conn, id stringuuid.StringUUID, rdb *goredis.Client) *Client {
	return &Client{
		conn:             c,
		MatchmakingMsgCh: make(chan matchmake.MatchResponse),
		GameMsgCh:        make(chan game.ServerResponse),
		ID:               id,
		rdb:              rdb,
	}
}

func (c *Client) writePump(ctx context.Context, cf context.CancelFunc) {
	var bytes []byte
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.MatchmakingMsgCh:
			bytes, err = json.Marshal(msg)
			if err != nil {
				fmt.Printf("failed to marshal incoming matchmaking msg - %v\n", err)
				continue
			}
		case msg := <-c.GameMsgCh:
			bytes, err = json.Marshal(msg)
			if err != nil {
				fmt.Printf("failed to marshal incoming matchmaking msg - %v\n", err)
				continue
			}
		}
		err = c.conn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			fmt.Println("failed to write to websocket", err)
			cf()
			return
		}
	}
}

func (c *Client) readPump(ctx context.Context, cf context.CancelFunc) {
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
				cf()
				return
			}
			c.ForwardMessageToService(bm, ctx, cf) // Pass context and cancelFunc
		case err := <-errChan:
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("client %s: websocket unexpectedly closed\n", c.ID)
			} else {
				fmt.Printf("client %s: websocket encountered an error: %v\n", c.ID, err)
			}
			cf()
			return
		}
	}
}

func (c *Client) ForwardMessageToService(msg BaseMessage, ctx context.Context, cf context.CancelFunc) {
	switch msg.Type {
	case MessageTypeGameplay:
		var action game.ClientActionBase

		json.Unmarshal(msg.Payload, &action)
		_, err := c.rdb.XAdd(ctx, &goredis.XAddArgs{
			Stream: redis.GameClientActionStream(action.GameID),
			Values: action,
			ID:     "*",
		}).Result()
		if err != nil {
			fmt.Printf("error forwarding gameplay message %v\n", err)
		}
	case MessageTypeMatchmaking:
		var matchReq matchmake.MatchRequest
		json.Unmarshal(msg.Payload, &matchReq)
		matchReq.UserId = c.ID
		_, err := c.rdb.XAdd(ctx, &goredis.XAddArgs{
			Stream: redis.MatchmakeRequestStream,
			Values: matchReq,
			ID:     "*",
		}).Result()
		if err != nil {
			fmt.Printf("error forwarding matchmake request %v\n", err)
		}
	default:
		fmt.Println("unexpected gateway.MessageType")
		cf()
	}
}

func (c *Client) listenToRedis(ctx context.Context, cf context.CancelFunc) {

	// read from matchmake response stream
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Printf("waiting to receive match for id=%s\n", c.ID)
				entries, err := c.rdb.XRead(ctx, &goredis.XReadArgs{
					Streams: []string{redis.MatchFoundStream(c.ID), "$"},
					Count:   1,
					Block:   0,
				}).Result()
				if err != nil {
					fmt.Printf("failed to read from match found stream - %v\n", err)
					continue
				}
				if len(entries) == 0 || len(entries[0].Messages) == 0 {
					fmt.Printf("read from match found stream zero results")
					continue
				}
				data := entries[0].Messages[0].Values

				var res matchmake.MatchResponse
				err = interfacestruct.Structify(data, &res)
				if err != nil {
					fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", data, err)
				}
				fmt.Printf("received new match response - %v\n", res)
				c.MatchmakingMsgCh <- res
			}
		}
	}()

	// read from server response stream
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				entries, err := c.rdb.XRead(ctx, &goredis.XReadArgs{
					Streams: []string{redis.GameServerResponseStream(c.ID), "$"},
					Count:   1,
					Block:   0,
				}).Result()
				if err != nil {
					fmt.Printf("failed to read from game server response stream - %v\n", err)
				}
				data := entries[0].Messages[0].Values
				if len(entries) == 0 || len(entries[0].Messages) == 0 {
					fmt.Printf("read from game server response stream with zero results")
					continue
				}

				var res game.ServerResponse
				err = interfacestruct.Structify(data, &res)
				if err != nil {
					fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", data, err)
				}
				c.GameMsgCh <- res
			}
		}
	}()
}
