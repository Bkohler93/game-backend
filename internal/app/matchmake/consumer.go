package matchmake

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/redis/go-redis/v9"
)

type RedisMatchmakingServerMessageConsumer struct {
	*transport.RedisMessageConsumer
}

func NewRedisMatchmakingServerMessageConsumer(ctx context.Context, rdb *redis.Client, consumer string) (*RedisMatchmakingServerMessageConsumer, error) {
	r := &RedisMatchmakingServerMessageConsumer{}
	stream := rediskeys.MatchmakingServerMessageStream
	consumerGroup := rediskeys.MatchmakingServerMessageCGroup
	c, err := transport.NewRedisMessageConsumer(ctx, rdb, stream, consumerGroup, consumer)
	if err != nil {
		return r, err
	}
	r.RedisMessageConsumer = c

	return r, nil
}
