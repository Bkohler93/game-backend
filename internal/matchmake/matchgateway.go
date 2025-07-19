package matchmake

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/redis/go-redis/v9"
)

type MatchGateway struct {
	rdb    *RedisClient
	addr   string
	conn   *websocket.Conn
	userId string
}

type RedisClient struct {
	*redis.Client
}

type Match struct {
	UserOne string
	UserTwo string
}

func (rc *RedisClient) RequestMatch(req MatchRequest, ctx context.Context) {
	_, err := rc.HSet(ctx, "user_pool:"+req.UserId, req).Result()
	if err != nil {
		fmt.Printf("failed to request match - %v\n", err)
		return
	}

	_, err = rc.XAdd(ctx, &redis.XAddArgs{
		Stream: "matchmake:signal",
		Values: map[string]interface{}{
			"hello": "there",
		}, //TODO add specifier to tell matchmaker what keys to pull
		ID: "*",
	}).Result()
	if err != nil {
		fmt.Printf("error signaling to start matchmaking - %v\n", err)
	}
}

const (
	pongWait = 60 * time.Second

	maxMessageSize = 512
)

// Todo change Id's to uuid.UUID

func NewMatchGateway(addr, redisAddr string) MatchGateway {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
	rc := RedisClient{rdb}

	return MatchGateway{
		rdb:  &rc,
		addr: addr,
	}
}

func (g *MatchGateway) Start(ctx context.Context) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		g.InitializeWebsocket(w, r)
		defer g.conn.Close()
		req, err := g.ReadRequest()
		if err != nil {
			fmt.Printf("error trying to read match request - %v\n", err)
			return
		}
		g.userId = req.UserId

		g.rdb.RequestMatch(req, ctx)

		g.waitForMatch(ctx)
	})

	fmt.Printf("listening on %s\n", g.addr)
	err := http.ListenAndServe(":"+g.addr, nil)
	if err != nil {
		log.Fatalf("error creating server - %v", err)
	}
}

func (g *MatchGateway) ReadRequest() (MatchRequest, error) {
	var req MatchRequest
	_, bytes, err := g.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			return req, err
		}
	}
	fmt.Println("received bytes=", bytes)
	err = json.Unmarshal(bytes, &req)
	if err != nil {
		return req, err
	}
	return req, nil
}

func (g *MatchGateway) InitializeWebsocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("could not upgrade to ws - %s\n", err)
		return
	}

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	g.conn = conn
}

func (g *MatchGateway) waitForMatch(ctx context.Context) {
	for {
		entries, err := g.rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{"match:made", "$"},
			Count:   1,
			Block:   0,
		}).Result()
		if err != nil {
			fmt.Printf("failed to read from matchmake:signal stream - %v\n", err)
			continue
		}
		res := entries[0].Messages[0].Values

		var match MatchResponse
		err = mapstructure.Decode(res, &match)
		if err != nil {
			fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", res, err)
			continue
		}

		if match.UserOneId == g.userId || match.UserTwoId == g.userId {
			bytes, err := json.Marshal(match)
			if err != nil {
				fmt.Printf("failed to encode match data - %v", err)
				break
			}
			g.conn.WriteMessage(websocket.TextMessage, bytes)
			break
		}
	}
}
