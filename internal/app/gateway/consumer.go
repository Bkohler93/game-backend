package gateway

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

type RedisMatchmakingClientMessageConsumerFactory struct {
	rdb *redis.Client
}

func NewRedisMatchmakingClientMessageConsumerFactory(rdb *redis.Client) *RedisMatchmakingClientMessageConsumerFactory {
	return &RedisMatchmakingClientMessageConsumerFactory{rdb}
}

func (r *RedisMatchmakingClientMessageConsumerFactory) CreateGroupConsumer(ctx context.Context, consumer string) (transport.MessageGroupConsumer, error) {
	stream := rediskeys.MatchmakingClientMessageStream(uuidstring.ID(consumer))
	consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(consumer))
	return transport.NewRedisMessageGroupConsumer(ctx, r.rdb, stream, consumerGroup, consumer)
}
