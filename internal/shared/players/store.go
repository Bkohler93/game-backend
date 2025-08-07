package players

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

type TrackerStore interface {
	Set(ctx context.Context, playerId, roomId uuidstring.ID) error
	Get(ctx context.Context, playerId uuidstring.ID) (uuidstring.ID, error)
}

type RedisTrackerStore struct {
	rdb *redis.Client
}

func NewRedisPlayerTrackerStore(rdb *redis.Client) *RedisTrackerStore {
	return &RedisTrackerStore{
		rdb: rdb,
	}
}

func playerKey(playerId uuidstring.ID) string {
	return rediskeys.PlayerString(playerId)
}

func (r *RedisTrackerStore) Set(ctx context.Context, playerId, roomId uuidstring.ID) error {
	key := playerKey(playerId)
	return r.rdb.Set(ctx, key, roomId, 0).Err()
}

func (r *RedisTrackerStore) Get(ctx context.Context, playerId uuidstring.ID) (uuidstring.ID, error) {
	key := playerKey(playerId)
	str, err := r.rdb.Get(ctx, key).Result()
	id := uuidstring.ID(str)
	return id, err
}
