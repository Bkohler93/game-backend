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

	redisWriteClient, err := redisutils.NewRedisMatchmakeClient(ctx)
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
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	streamListener := transport.NewRedisStreamListener(ctx, redisClient, []string{rediskeys.MatchmakingClientMessageStreamDestination(hostname), rediskeys.GameClientMessageStreamDestination(hostname)})

	transportFactory := &gateway.TransportFactory{
		MatchmakingClientMsgConsumerBuilder: func(ctx context.Context, clientId string) (transport.MessageConsumer, error) {
			// consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(clientId))
			// return transport.NewRedisMessageGroupConsumer(ctx, redisClient, stream, consumerGroup, clientId)
			return streamListener.AddConsumer(clientId)
		},
		MatchmakingClientMsgConsumerDestroyer: func(clientId string) {
			streamListener.RemoveConsumer(clientId)
		},
		GameClientMsgConsumerBuilder: func(ctx context.Context, clientId string) (transport.MessageConsumer, error) {
			//destinationID := rediskeys.GameClientMessageStreamDestination(uuidstring.ID(clientId))
			// consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(clientId))
			// return transport.NewRedisMessageGroupConsumer(ctx, redisClient, stream, consumerGroup, clientId)
			return streamListener.AddConsumer(clientId)
		},
		GameClientMsgConsumerDestroyer: func(clientId string) {
			//destinationID := rediskeys.GameClientMessageStreamDestination(clientId)
			streamListener.RemoveConsumer(clientId)
		},
		MatchmakingServerMsgProducerBuilder: func() transport.MessageProducer {
			return transport.NewRedisMessageProducer(redisWriteClient, rediskeys.MatchmakingServerMessageStream)
		},
		GameplayServerMsgProducerBuilder: func() transport.DynamicMessageProducer {
			return transport.NewRedisDynamicMessageProducer(redisWriteClient, rediskeys.GameServerMessageStreamDestination)
		},
	}

	g := gateway.NewGateway(port, roomRepository, transportFactory)

	g.Start(ctx)
}
