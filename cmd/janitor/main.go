package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bkohler93/game-backend/internal/app/janitor"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
)

func main() {
	utils.LoadEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	redisClient, err := redisutils.NewRedisClient(ctx)
	if err != nil {
		panic(err)
	}
	matchmakingTaskStore, err := taskcoordinator.NewRedisMatchmakingTaskStore(redisClient)
	if err != nil {
		panic(err)
	}

	taskCoordinator := taskcoordinator.NewMatchmakingTaskCoordinator(matchmakingTaskStore)

	j := janitor.New(taskCoordinator)
	j.Start(ctx)
}
