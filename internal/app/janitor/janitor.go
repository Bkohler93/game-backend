package janitor

import (
	"context"
	"fmt"
	"time"

	"github.com/bkohler93/game-backend/internal/matchmake/taskcoordinator"
	"github.com/bkohler93/game-backend/internal/utils/redisutils/rediskeys"
)

var (
	luaScriptBasePath                = "../../db/redis/scripts"
	staleMatchmakingToPendingLuaPath = fmt.Sprintf("%s/stale_matchmaking_to_pending.lua", luaScriptBasePath)
)

type Janitor struct {
	matchmakingCoordinator *taskcoordinator.MatchmakingTaskCoordinator
	//rdb     *redis.Client
	//scripts map[string]*redis.Script
}

func New(matchmakingCoordinator *taskcoordinator.MatchmakingTaskCoordinator) *Janitor {
	//addPlayerToRoomSrc, err := loadLuaSrc(staleMatchmakingToPendingLuaPath)
	j := &Janitor{
		matchmakingCoordinator: matchmakingCoordinator,
	}
	//if err != nil {
	//	fmt.Printf("failed to load lua src from '%s' with error - %v", staleMatchmakingToPendingLuaPath, err)
	//	return j
	//}
	//j.scripts[staleMatchmakingToPendingLuaPath] = redis.NewScript(addPlayerToRoomSrc)
	return j
}

const (
	cleanupFrequency  = 5 * time.Second
	cleanupDuration   = 2 * time.Second
	allowableIdleTime = 5 * time.Second
)

func (j *Janitor) Start(ctx context.Context) {
	t := time.NewTicker(cleanupFrequency)
	for {
		select {
		case <-t.C:
			innerCtx, cancel := context.WithTimeout(ctx, cleanupDuration)
			j.CleanupServerMessages(innerCtx)
			j.CleanupMatchmakeTasksInProgress(innerCtx)
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func (j *Janitor) CleanupServerMessages(ctx context.Context) {
	matchmakingStream := rediskeys.MatchmakingServerMessageStream
	matchmakingConsumerGroup := rediskeys.MatchmakingServerMessageCGroup
	j.cleanConsumerGroup(ctx, matchmakingStream, matchmakingConsumerGroup)

	//TODO once Game server is running should run it for the `game:server_message` stream consumer group
}

func (j *Janitor) cleanConsumerGroup(ctx context.Context, stream string, consumerGroup string) {
	//TODO abstract consumer group/stream logic, still have to find where to create consumer groups
	//consumers, err := j.rdb.XInfoConsumers(ctx, stream, consumerGroup).Result()
	//if err != nil {
	//	fmt.Printf("failed to get info on %s consumer group - %v", consumerGroup, err)
	//	return
	//}
	//
	//activeConsumers := make([]string, 0, len(consumers))
	//
	//utils.SliceForeachContext(ctx, consumers, func(ctx context.Context, item redis.XInfoConsumer) {
	//	if item.Pending > 0 && item.Idle > allowableIdleTime {
	//		_, err := j.rdb.XGroupDelConsumer(ctx, stream, consumerGroup, item.Name).Result()
	//		if err != nil {
	//			fmt.Printf("failed to delete %s from %s group with error - %v", item.Name, consumerGroup, err)
	//		}
	//		return
	//	}
	//	activeConsumers = append(activeConsumers, item.Name)
	//})
	//idx := -1
	//getConsumer := func() string {
	//	idx++
	//	if idx == len(activeConsumers) {
	//		idx = 0
	//	}
	//	return activeConsumers[idx]
	//}
	//start := "0-0"
	//var ids []string
	//for {
	//	cname := getConsumer()
	//	ids, start, err = j.rdb.XAutoClaimJustID(
	//		ctx,
	//		&redis.XAutoClaimArgs{
	//			Stream:   stream,
	//			Group:    consumerGroup,
	//			MinIdle:  allowableIdleTime,
	//			Start:    start,
	//			Count:    1,
	//			Consumer: cname,
	//		},
	//	).Result()
	//	if err != nil {
	//		fmt.Printf("failed to AutoClaim on message from %s group with error - %v", consumerGroup, err)
	//	}
	//	fmt.Printf("%s autoclaimed these ids: %v\n", cname, ids)
	//	if start == "0-0" {
	//		break
	//	}
	//}
}

func (j *Janitor) CleanupMatchmakeTasksInProgress(ctx context.Context) {
	maxTime := time.Now().Add(allowableIdleTime * -1).Unix()
	_, err := j.matchmakingCoordinator.ReclaimStaleInProgressTasks(ctx, maxTime)
	if err != nil {
		fmt.Printf("failed to reclaim old inprogress tasks with err - %v\n", err)
	}
	//key := rediskeys.MatchmakeTaskInProgressSortedSet
	//
	//staleRoomIds, err := j.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
	//	Min: "-",
	//	Max: maxTime,
	//}).Result()
	//if err != nil {
	//	fmt.Printf("failed to retrieve stale room id's - %v\n", err)
	//	return
	//}
	//
	//for _, rmId := range staleRoomIds {
	//	//TODO lua script for atomic remove member from InProgress Set into Pending Set with time Now for processing
	//	roomId := stringuuid.StringUUID(rmId)
	//	inProgressKey := rediskeys.MatchmakeTaskInProgressKey(roomId)
	//	pendingKey := rediskeys.MatchmakeTaskPendingKey(roomId)
	//	n := time.Now().Unix()
	//	err := j.scripts[staleMatchmakingToPendingLuaPath].Run(ctx, j.rdb, []string{inProgressKey, pendingKey}, n).Err()
	//	if err != nil {
	//		fmt.Printf("failed to run script at %s - %v\n", staleMatchmakingToPendingLuaPath, err)
	//		return
	//	}
	//}
}
