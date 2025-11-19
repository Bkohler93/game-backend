package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bkohler93/game-backend/internal/app/game"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
)

func main() {
	utils.LoadEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	redisClient, err := redisutils.NewRedisMatchmakeClient(ctx)
	if err != nil {
		panic(err)
	}
	gameTaskStore, err := taskcoordinator.NewRedisGameTaskStore(redisClient)
	if err != nil {
		panic(err)
	}
	roomStore, err := room.NewRedisRoomStore(redisClient)
	if err != nil {
		panic(err)
	}

	taskCoordinator := taskcoordinator.NewGameTaskCoordinator(gameTaskStore)
	// redisStreamListener := transport.NewRedisStreamListener(ctx, redisClient)

	tf := &game.TransportFactory{
		GameServerMsgConsumerBuilder: func(ctx context.Context, roomID string) (transport.MessageConsumer, error) {
			//stream := rediskeys.GameServerMessageStream(uuidstring.ID(roomID))
			//consumerGroup := rediskeys.GameServerMessageCGroup(uuidstring.ID(roomID))
			//return transport.NewRedisMessageGroupConsumer(ctx, redisClient, stream, consumerGroup, roomID)
			// return transport.NewRedisMessageConsumer(ctx, redisStreamListener, stream), nil
			// return redisStreamListener.AddConsumer(stream), nil

			listener := transport.NewRedisStreamListener(ctx, redisClient, []string{rediskeys.GameServerMessageStream})
			return listener.AddConsumer(roomID)
		},
		GameClientMsgProducerBuilder: func() transport.DynamicMessageProducer {
			return transport.NewRedisDynamicMessageProducer(redisClient, func(targetHostname string) string {
				return rediskeys.GameClientMessageStreamDestination(targetHostname)
			})
		},
	}

	g := game.New(taskCoordinator, roomStore, tf)
	g.Start(ctx)
}
