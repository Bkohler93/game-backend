package taskcoordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils/files"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

type RedisGameTaskStore struct {
	rdb *redis.Client
	lua map[string]*redis.Script
}

func NewRedisGameTaskStore(rdb *redis.Client) (*RedisGameTaskStore, error) {
	r := &RedisGameTaskStore{
		rdb: rdb,
		lua: make(map[string]*redis.Script),
	}
	zPopOldestFromSortedSetSrc, err := files.GetLuaScript(files.LuaZPopOldestFromSortedSet)
	if err != nil {
		return r, fmt.Errorf("failed to load lua src from '%s' with error - %v", files.LuaZPopOldestFromSortedSet, err)
	}

	r.lua[files.LuaZPopOldestFromSortedSet] = redis.NewScript(zPopOldestFromSortedSetSrc)
	return r, nil
}

func (s *RedisGameTaskStore) AddNewGameTask(ctx context.Context, roomID uuidstring.ID) error {
	return s.rdb.ZAdd(ctx, rediskeys.NewGameSortedSetKey, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: roomID.String(),
	}).Err()
}

func (s *RedisGameTaskStore) RemoveOldestNewGameTask(ctx context.Context) (uuidstring.ID, error) {
	res, err := s.lua[files.LuaZPopOldestFromSortedSet].Run(ctx, s.rdb, []string{rediskeys.NewGameSortedSetKey}, time.Now().Unix()).Result()
	if err != nil {
		if err.Error() == "NO_AVAILABLE_TASK" {
			return "", ErrNoTasksAvailable
		}
		return "", err
	}
	roomID := res.(string)
	return uuidstring.ID(roomID), nil
}
