package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bkohler93/game-backend/internal/matchmake"
	"github.com/bkohler93/game-backend/internal/redis"
	"github.com/bkohler93/game-backend/pkg/interfacestruct"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/gorilla/websocket"
	goredis "github.com/redis/go-redis/v9"
)

const (
	pongWait = 60 * time.Second

	maxMessageSize = 512
)

type Gateway struct {
	rdb  *goredis.Client
	addr string
	hub  Hub
}

func NewGateway(addr, redisAddr string) Gateway {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	return Gateway{
		rdb:  rdb,
		addr: addr,
		hub:  *NewHub(),
	}
}

func (g *Gateway) Start(ctx context.Context) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ctx, cf := context.WithCancel(ctx)
		client, err := g.InitializeWebsocket(w, r)
		if err != nil {
			fmt.Printf("failed to initialize client websocket - %v\n", err)
			cf()
			return
		}
		defer client.conn.Close()

		g.hub.RegisterCh <- client

		go client.writePump(ctx, cf)
		go client.readPump(ctx, cf)

		go client.listenToRedis(ctx, cf)

		<-ctx.Done()
		fmt.Println("Websocket connection closed")
		g.hub.UnregisterCh <- client
	})

	go g.listenForMatches(ctx)

	fmt.Printf("listening on %s for new matchmaking requests from clients\n", g.addr)
	err := http.ListenAndServe("0.0.0.0:"+g.addr, nil)
	if err != nil {
		log.Fatalf("error creating server - %v", err)
	}
}

// func (m *Gateway) RequestMatch(req matchmake.MatchRequest, ctx context.Context) {
// 	data, err := interfacestruct.Interfacify(req)
// 	if err != nil {
// 		fmt.Printf("failed to turn match request into interface - %v\n", err)
// 		return
// 	}
// 	_, err = m.rdb.XAdd(ctx, &redis.XAddArgs{
// 		Stream: "matchmake:signal",
// 		Values: data, //TODO add specifier to tell matchmaker what keys to pull
// 		ID:     "*",
// 	}).Result()
// 	if err != nil {
// 		fmt.Printf("error signaling to start matchmaking - %v\n", err)
// 	}
// }

// Todo change Id's to uuid.UUID

func (m *Gateway) SaveUserId(req matchmake.MatchRequest, ctx context.Context) context.Context {
	return context.WithValue(ctx, "userId", req.UserId)
}

func (m *Gateway) GetUserId(ctx context.Context) stringuuid.StringUUID {
	id := ctx.Value("userId")
	return id.(stringuuid.StringUUID)
}

func (g *Gateway) InitializeWebsocket(w http.ResponseWriter, r *http.Request) (*Client, error) {
	var c *Client
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return c, err
	}

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	id := stringuuid.NewStringUUID()
	c = NewClient(conn, id, g.rdb)

	return c, nil
}

func (g *Gateway) listenForMatches(ctx context.Context) {
	for {
		fmt.Println("waiting for match to come through stream")
		entries, err := g.rdb.XRead(ctx, &goredis.XReadArgs{
			Streams: []string{redis.MatchfoundStream, "$"},
			Count:   1,
			Block:   0,
		}).Result()
		if err != nil {
			fmt.Printf("failed to read from matchmake:found stream - %v\n", err)
			continue
		}
		res := entries[0].Messages[0].Values

		var match matchmake.MatchResponse
		err = interfacestruct.Structify(res, &match)
		if err != nil {
			fmt.Printf("failed to scan {%v} into new MatchResponse - %v\n", res, err)
			continue
		}
		g.hub.Clients[match.UserOneId].MatchmakingMsgCh <- match
		g.hub.Clients[match.UserTwoId].MatchmakingMsgCh <- match
		// if match.UserOneId == userId || match.UserTwoId == userId {
		// 	bytes, err := json.Marshal(match)
		// 	if err != nil {
		// 		fmt.Printf("failed to encode match data - %v", err)
		// 		break
		// 	}
		// 	g.conn.WriteMessage(websocket.TextMessage, bytes)
		// 	break
		// }
	}
}
