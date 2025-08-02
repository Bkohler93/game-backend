package main

import (
	"context"

	"github.com/bkohler93/game-backend/internal/app/matchmake"
	"github.com/bkohler93/game-backend/internal/shared/players"
	"github.com/bkohler93/game-backend/internal/shared/room"
	"github.com/bkohler93/game-backend/internal/shared/taskcoordinator"
	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
)

var ctx = context.Background()

func main() {
	utils.LoadEnv()

	redisClient, err := redisutils.NewRedisClient(ctx)
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

	mmClientMsgProducer := matchmake.NewMatchmakingClientMessageRedisProducer(redisClient)
	roomRepository := room.NewRepository(roomStore)
	playerRepository := players.NewRepository(playerTrackerStore)
	matchmakingTaskCoordinator := taskcoordinator.NewMatchmakingTaskCoordinator(matchmakingTaskStore)

	m := matchmake.Matchmaker{
		MatchmakingClientMessageProducer: mmClientMsgProducer,
		RoomRepository:                   roomRepository,
		PlayerRepository:                 playerRepository,
		MatchmakingTaskCoordinator:       matchmakingTaskCoordinator,
	}
	m.Start(ctx)
}
