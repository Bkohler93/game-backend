package gateway

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/redis/go-redis/v9"
)

type RedisMatchmakingServerMessageProducer struct {
	*transport.RedisMessageProducer
}

func NewRedisMatchmakingServerMessageProducer(rdb *redis.Client) *RedisMatchmakingServerMessageProducer {
	p := transport.NewRedisMessageProducer(rdb, rediskeys.MatchmakingServerMessageStream)
	return &RedisMatchmakingServerMessageProducer{p}
}

type RedisMatchmakingServerMessageProducerFactory struct {
	rdb *redis.Client
}

func NewRedisMatchmakingServerMessageProducerFactory(rdb *redis.Client) *RedisMatchmakingServerMessageProducerFactory {
	return &RedisMatchmakingServerMessageProducerFactory{rdb: rdb}
}

func (r *RedisMatchmakingServerMessageProducerFactory) CreateProducer(ctx context.Context) (transport.MessageProducer, error) {
	p := transport.NewRedisMessageProducer(r.rdb, rediskeys.MatchmakingServerMessageStream)
	producer := &RedisMatchmakingServerMessageProducer{p}
	return producer, nil
}
