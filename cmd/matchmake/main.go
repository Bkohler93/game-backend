package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bkohler93/game-backend/internal/app/matchmake"
	"github.com/bkohler93/game-backend/internal/shared/players"
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
	roomStore, err := room.NewRedisRoomStore(redisClient)
	if err != nil {
		panic(err)
	}
	playerTrackerStore := players.NewRedisPlayerTrackerStore(redisClient)

	matchmakingTaskStore, err := taskcoordinator.NewRedisMatchmakingTaskStore(redisClient)
	if err != nil {
		panic(err)
	}
	gameTaskStore, err := taskcoordinator.NewRedisGameTaskStore(redisClient)
	if err != nil {
		panic(err)
	}
	// serverId := uuidstring.NewID()

	roomRepository, err := room.NewRepository(ctx, roomStore)
	if err != nil {
		panic(err)
	}
	playerRepository := players.NewRepository(playerTrackerStore)
	matchmakingTaskCoordinator := taskcoordinator.NewMatchmakingTaskCoordinator(matchmakingTaskStore)
	gameTaskCoordinator := taskcoordinator.NewGameTaskCoordinator(gameTaskStore)

	matchmakingClientMessageProducer := matchmake.NewRedisClientMessageProducer(redisClient)

	redisStreamListener := transport.NewRedisStreamListener(ctx, redisClient)
	// matchmakingServerMessageConsumer, err := matchmake.NewRedisMatchmakingServerMessageConsumer(ctx, redisClient, serverId.String())

	matchmakeWorkerNotifier := matchmake.NewRedisWorkerNotifierBroadcastProducer(redisClient)
	matchmakeWorkerNotifyListener := matchmake.NewRedisWorkerNotifierListener(redisClient)
	if err != nil {
		panic(err)
	}
	// bus := matchmake.NewBus(matchmakingServerMessageConsumer, matchmakingClientMessageProducer, matchmakeWorkerNotifier, matchmakeWorkerNotifyListener)
	bus := matchmake.NewBus(redisStreamListener.AddConsumer(rediskeys.MatchmakingServerMessageStream), matchmakingClientMessageProducer, matchmakeWorkerNotifier, matchmakeWorkerNotifyListener)

	m := matchmake.Matchmaker{
		TransportBus:               bus,
		RoomRepository:             roomRepository,
		PlayerRepository:           playerRepository,
		MatchmakingTaskCoordinator: matchmakingTaskCoordinator,
		GameTaskCoordinator:        gameTaskCoordinator,
	}
	m.Start(ctx)
}
