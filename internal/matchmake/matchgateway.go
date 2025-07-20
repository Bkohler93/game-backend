package matchmake

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/pkg/interfacestruct"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type MatchGateway struct {
	rdb   *redis.Client
	addr  string
	conn  *websocket.Conn
	msgCh chan ([]byte)
}

type Match struct {
	UserOne string
	UserTwo string
}

func (m *MatchGateway) RequestMatch(req MatchRequest, ctx context.Context) {
	data, err := interfacestruct.Interfacify(req)
	if err != nil {
		fmt.Printf("failed to turn match request into interface - %v\n", err)
		return
	}
	_, err = m.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "matchmake:signal",
		Values: data, //TODO add specifier to tell matchmaker what keys to pull
		ID:     "*",
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

	return MatchGateway{
		rdb:   rdb,
		addr:  addr,
		msgCh: make(chan []byte),
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
		if req.UserId.UUID() == uuid.Nil {
			req.UserId = stringuuid.NewUserId()
		}
		ctx = g.SaveUserId(req, ctx)
		g.RequestMatch(req, ctx)

		g.waitForMatch(ctx)
		time.Sleep(time.Second * 2)
	})

	fmt.Printf("listening on %s\n", g.addr)
	err := http.ListenAndServe("0.0.0.0:"+g.addr, nil)
	if err != nil {
		log.Fatalf("error creating server - %v", err)
	}
}

func (m *MatchGateway) SaveUserId(req MatchRequest, ctx context.Context) context.Context {
	return context.WithValue(ctx, "userId", req.UserId)
}

func (m *MatchGateway) GetUserId(ctx context.Context) stringuuid.UserId {
	id := ctx.Value("userId")
	return id.(stringuuid.UserId)
}

func (g *MatchGateway) ReadRequest() (MatchRequest, error) {
	var req MatchRequest
	_, bytes, err := g.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			return req, err
		}
	}
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

	go g.writePump()
}

func (g *MatchGateway) writePump() {
	for {
		msg := <-g.msgCh
		g.conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func (g *MatchGateway) waitForMatch(ctx context.Context) {
	userId := g.GetUserId(ctx)
	for {
		fmt.Println("waiting for match to come through stream")
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
		err = interfacestruct.Structify(res, &match)
		if err != nil {
			fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", res, err)
			continue
		}

		if match.UserOneId == userId || match.UserTwoId == userId {
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
