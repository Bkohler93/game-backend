package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bkohler93/game-backend/internal/app/gateway"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

func main() {
	utils.LoadEnv()
	port := os.Getenv("WEBSOCKET_PORT")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	redisClient, err := redisutils.NewRedisMatchmakeClient(ctx)
	if err != nil {
		panic(err)
	}

	roomStore, err := room.NewRedisRoomStore(redisClient)
	if err != nil {
		panic(err)
	}
	roomRepository, err := room.NewRepository(ctx, roomStore)
	if err != nil {
		panic(err)
	}

	transportFactory := &gateway.TransportFactory{
		MatchmakingClientMsgConsumerBuilder: func(ctx context.Context, clientId string) (transport.MessageConsumer, error) {
			stream := rediskeys.MatchmakingClientMessageStream(uuidstring.ID(clientId))
			consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(clientId))
			return transport.NewRedisMessageGroupConsumer(ctx, redisClient, stream, consumerGroup, clientId)
		},
		GameClientMsgConsumerBuilder: func(ctx context.Context, clientId string) (transport.MessageConsumer, error) {
			stream := rediskeys.GameClientMessageStream(uuidstring.ID(clientId))
			consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(clientId))
			return transport.NewRedisMessageGroupConsumer(ctx, redisClient, stream, consumerGroup, clientId)
		},
		MatchmakingServerMsgProducerBuilder: func() transport.MessageProducer {
			return transport.NewRedisMessageProducer(redisClient, rediskeys.MatchmakingServerMessageStream)
		},
		GameplayServerMsgProducerBuilder: func() transport.DynamicMessageProducer {
			return transport.NewRedisDynamicMessageProducer(redisClient, rediskeys.GameServerMessageStream)
		},
	}

	g := gateway.NewGateway(port, roomRepository, transportFactory)

	g.Start(ctx)
}
