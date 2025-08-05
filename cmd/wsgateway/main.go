package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bkohler93/game-backend/internal/app/gateway"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
)

func main() {
	utils.LoadEnv()
	port := os.Getenv("PORT")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisClient, err := redisutils.NewRedisClient(ctx)
	if err != nil {
		panic(err)
	}

	roomStore, err := room.NewRedisRoomStore(redisClient)
	if err != nil {
		panic(err)
	}
	roomRepository := room.NewRepository(roomStore)
	matchmakingClientMsgConsumerFactory := gateway.NewRedisMatchmakingClientMessageConsumerFactory(redisClient)
	matchmakingServerMsgProducer := gateway.NewRedisMatchmakingServerMessageProducer(redisClient)

	g := gateway.NewGateway(port, roomRepository, matchmakingClientMsgConsumerFactory, matchmakingServerMsgProducer)

	g.Start(ctx)
}
