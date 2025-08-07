package taskcoordinator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

var (
	luaScriptBasePath                   = "../../../db/redis/scripts"
	staleMatchmakingToPendingLuaPath    = fmt.Sprintf("%s/stale_matchmaking_to_pending.lua", luaScriptBasePath)
	claimPendingMoveToInProgressLuaPath = fmt.Sprintf("%s/claim_pending_move_to_inprogress.lua", luaScriptBasePath)
)

const (
	NoPendingTasksAvailableErr = "no available task in pending set"
	UnexpectedRedisResultErr   = "unexpected redis result type"
)

type RedisMatchmakingTaskStore struct {
	rdb *redis.Client
	lua map[string]*redis.Script
}

func NewRedisMatchmakingTaskStore(rdb *redis.Client) (*RedisMatchmakingTaskStore, error) {
	r := &RedisMatchmakingTaskStore{
		rdb: rdb,
		lua: make(map[string]*redis.Script),
	}
	staleMatchmakingToPendingSrc, err := utils.LoadLuaSrc(staleMatchmakingToPendingLuaPath)
	if err != nil {
		return r, fmt.Errorf("failed to load lua src from '%s' with error - %v", staleMatchmakingToPendingLuaPath, err)
	}
	claimPendingMoveToInProgressSrc, err := utils.LoadLuaSrc(claimPendingMoveToInProgressLuaPath)
	if err != nil {
		return r, fmt.Errorf("failed to load lua src from '%s' with error - %v", claimPendingMoveToInProgressLuaPath, err)
	}

	r.lua[claimPendingMoveToInProgressLuaPath] = redis.NewScript(claimPendingMoveToInProgressSrc)
	r.lua[staleMatchmakingToPendingLuaPath] = redis.NewScript(staleMatchmakingToPendingSrc)
	return r, nil
}

func (r *RedisMatchmakingTaskStore) AddInProgressTask(ctx context.Context, roomID uuidstring.ID, score int64) error {
	return r.rdb.ZAdd(ctx, rediskeys.MatchmakeTaskInProgressSortedSetKey, redis.Z{
		Score:  float64(score),
		Member: roomID,
	}).Err()
}

func (r *RedisMatchmakingTaskStore) AddPendingTask(ctx context.Context, roomID uuidstring.ID, score int64) error {
	return r.rdb.ZAdd(ctx, rediskeys.MatchmakeTaskPendingSortedSetKey, redis.Z{
		Score:  float64(score),
		Member: roomID,
	}).Err()
}

func (r *RedisMatchmakingTaskStore) RemoveInProgressTask(ctx context.Context, roomID uuidstring.ID) error {
	return r.rdb.ZRem(ctx, rediskeys.MatchmakeTaskInProgressSortedSetKey, roomID).Err()
}

func (r *RedisMatchmakingTaskStore) RemovePendingTask(ctx context.Context, roomID uuidstring.ID) error {
	return r.rdb.ZRem(ctx, rediskeys.MatchmakeTaskPendingSortedSetKey, roomID).Err()
}

func (r *RedisMatchmakingTaskStore) GetStaleInProgressTasks(ctx context.Context, cutoff int64) ([]uuidstring.ID, error) {
	var roomIds []uuidstring.ID
	members, err := r.rdb.ZRangeByScore(ctx, rediskeys.MatchmakeTaskInProgressSortedSetKey, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%f", float64(cutoff)),
		Count: -1,
	}).Result()
	if err != nil {
		return roomIds, err
	}
	for _, m := range members {
		roomId := uuidstring.ID(m)
		roomIds = append(roomIds, roomId)
	}
	return roomIds, err
}

func (r *RedisMatchmakingTaskStore) MoveInProgressToPendingTask(ctx context.Context, roomID uuidstring.ID) error {
	t := time.Now().Unix()
	err := r.lua[staleMatchmakingToPendingLuaPath].Run(
		ctx,
		r.rdb,
		[]string{rediskeys.MatchmakeTaskInProgressSortedSetKey, rediskeys.MatchmakeTaskPendingSortedSetKey},
		roomID,
		t,
	).Err()
	if err.Error() == "MEMBER_NOT_FOUND" {
		return fmt.Errorf("failed to move non-existent in-progress member %s - %v", roomID, err)
	} else if err.Error() == "MEMBER_NOT_ADDED" {
		return fmt.Errorf("failed to add member %s with time %d - %v", roomID, t, err)
	}
	return err
}

func (r *RedisMatchmakingTaskStore) ClaimPendingTask(ctx context.Context) (uuidstring.ID, error) {
	now := time.Now().Unix()

	res, err := r.lua[claimPendingMoveToInProgressLuaPath].Run(
		ctx,
		r.rdb,
		[]string{
			rediskeys.MatchmakeTaskPendingSortedSetKey,
			rediskeys.MatchmakeTaskInProgressSortedSetKey,
		},
		fmt.Sprintf("%d", now),
	).Result()
	if err != nil {
		return "", err
	}

	taskID, ok := res.(string)
	if !ok {
		return "", errors.New(UnexpectedRedisResultErr)
	}
	return uuidstring.ID(taskID), nil
}
