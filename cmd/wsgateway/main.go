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
	port := os.Getenv("PORT")
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
		MatchmakingClientMsgConsumerBuilder: func(clientId string) transport.MessageGroupConsumer {
			stream := rediskeys.MatchmakingClientMessageStream(uuidstring.ID(clientId))
			consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(clientId))
			consumer, err := transport.NewRedisMessageGroupConsumer(ctx, redisClient, stream, consumerGroup, clientId)
			if err != nil {
				panic(err)
			}
			return consumer
		},
		SetupClientMsgConsumerBuilder: func(roomId string) transport.MessageConsumer {
			stream := rediskeys.SetupClientMessageStream(uuidstring.ID(roomId))
			return transport.NewRedisMessageConsumer(redisClient, stream)
		},
		GameplayClientMsgConsumerBuilder: func(roomId string) transport.MessageConsumer {
			stream := rediskeys.GameClientMessageStream(uuidstring.ID(roomId))
			return transport.NewRedisMessageConsumer(redisClient, stream)
		},
		MatchmakingServerMsgProducerBuilder: func() transport.MessageProducer {
			return transport.NewRedisMessageProducer(redisClient, rediskeys.MatchmakingServerMessageStream)
		},
		SetupServerMsgProducerBuilder: func() transport.DynamicMessageProducer {
			return transport.NewRedisDynamicMessageProducer(redisClient, rediskeys.SetupClientMessageStream)
		},
		GameplayServerMsgProducerBuilder: func() transport.DynamicMessageProducer {
			return transport.NewRedisDynamicMessageProducer(redisClient, rediskeys.GameClientMessageStream)
		},
	}
	//clientTransportBusFactory := client.NewClientTransportBusFactory(redisClient, func(clientId string) transport.MessageGroupConsumer {
	//	stream := rediskeys.MatchmakingClientMessageStream(uuidstring.ID(clientId))
	//	consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(clientId))
	//	consumer, err := transport.NewRedisMessageGroupConsumer(ctx, redisClient, stream, consumerGroup, clientId)
	//	if err != nil {
	//		panic(err)
	//	}
	//	return consumer
	//}, func() transport.MessageProducer {
	//	return transport.NewRedisMessageProducer(redisClient, rediskeys.MatchmakingServerMessageStream)
	//})

	g := gateway.NewGateway(port, roomRepository, transportFactory)

	g.Start(ctx)
}
