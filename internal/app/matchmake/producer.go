package matchmake

import (
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/redis/go-redis/v9"
)

type RedisMatchmakingClientMessageProducer struct {
	*transport.RedisDynamicMessageProducer
}

func NewMatchmakingClientMessageRedisProducer(rdb *redis.Client) *RedisMatchmakingClientMessageProducer {
	p := transport.NewRedisDynamicMessageProducer(rdb, rediskeys.MatchmakingClientMessageStream)
	m := &RedisMatchmakingClientMessageProducer{p}
	return m
}
