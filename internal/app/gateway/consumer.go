package gateway

import (
	"context"
	"fmt"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

type RedisMatchmakingClientMessageConsumerFactory struct {
	rdb *redis.Client
}

func (r *RedisMatchmakingClientMessageConsumerFactory) CreateConsumer(ctx context.Context, consumer string) (transport.MessageConsumer, error) {
	stream := rediskeys.MatchmakingClientMessageStream(stringuuid.StringUUID(consumer))
	consumerGroup := rediskeys.MatchmakingClientMessageCGroup(stringuuid.StringUUID(consumer))
	return NewRedisMatchmakingClientMessageConsumer(ctx, r.rdb, stream, consumer, consumerGroup)
}

//func (r *RedisMatchmakingClientMessageConsumerFactory) CreateConsumer(ctx context.Context, consumer string) (MatchmakingClientMessageConsumer, error) {
//}

//func (r *RedisMatchmakingClientMessageConsumerFactory) CreateConsumer(ctx context.Context, stream, consumer, consumerGroup string) (MatchmakingClientMessageConsumer, error) {
//	return NewRedisMatchmakingClientMessageConsumer(ctx, r.rdb, stream, consumer, consumerGroup)
//}

func NewRedisMatchmakingClientMessageConsumerFactory(rdb *redis.Client) *RedisMatchmakingClientMessageConsumerFactory {
	return &RedisMatchmakingClientMessageConsumerFactory{rdb: rdb}
}

type RedisMatchmakingClientMessageConsumer struct {
	*transport.RedisMessageConsumer
}

func NewRedisMatchmakingClientMessageConsumer(ctx context.Context, rdb *redis.Client, stream, consumer, consumerGroup string) (*RedisMatchmakingClientMessageConsumer, error) {
	var r *RedisMatchmakingClientMessageConsumer
	c, err := transport.NewRedisMessageConsumer(ctx, rdb, stream, consumerGroup, consumer)
	if err != nil {
		fmt.Printf("failed to initialize RedisMessageConsumer - %v", err)
	}
	r = &RedisMatchmakingClientMessageConsumer{
		RedisMessageConsumer: c,
	}
	return r, err
}
