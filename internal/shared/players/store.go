package players

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

type TrackerStore interface {
	Set(ctx context.Context, playerId, roomId stringuuid.StringUUID) error
	Get(ctx context.Context, playerId stringuuid.StringUUID) (stringuuid.StringUUID, error)
}

type RedisTrackerStore struct {
	rdb *redis.Client
}

func NewRedisPlayerTrackerStore(rdb *redis.Client) *RedisTrackerStore {
	return &RedisTrackerStore{
		rdb: rdb,
	}
}

func playerKey(playerId stringuuid.StringUUID) string {
	return rediskeys.PlayerString(playerId)
}

func (r *RedisTrackerStore) Set(ctx context.Context, playerId, roomId stringuuid.StringUUID) error {
	key := playerKey(playerId)
	return r.rdb.Set(ctx, key, roomId, 0).Err()
}

func (r *RedisTrackerStore) Get(ctx context.Context, playerId stringuuid.StringUUID) (stringuuid.StringUUID, error) {
	key := playerKey(playerId)
	str, err := r.rdb.Get(ctx, key).Result()
	id := stringuuid.StringUUID(str)
	return id, err
}
