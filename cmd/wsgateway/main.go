package main

import (
	"context"
	"os"

	"github.com/bkohler93/game-backend/internal/app/gateway"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
)

var ctx = context.Background()

func main() {
	utils.LoadEnv()
	port := os.Getenv("PORT")

	redisClient, err := redisutils.NewRedisClient(ctx)
	if err != nil {
		panic(err)
	}

	roomStore, err := room.NewRedisRoomStore(redisClient)
	if err != nil {
		panic(err)
	}
	roomRepository := room.NewRepository(roomStore)
	matchmakingMsgConsumerFactory := transport.NewRedisMatchmakingClientMessageConsumerFactory(redisClient)

	g := gateway.NewGateway(port, roomRepository, matchmakingMsgConsumerFactory)

	g.Start(ctx)
}
