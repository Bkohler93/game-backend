package matchmake

import (
	"context"
	"fmt"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/redis/go-redis/v9"
)

func NewRedisMatchmakingServerMessageConsumer(ctx context.Context, rdb *redis.Client, consumer string) (*transport.RedisMessageGroupConsumer, error) {
	stream := rediskeys.MatchmakingServerMessageStream
	consumerGroup := rediskeys.MatchmakingServerMessageCGroup
	consumer = fmt.Sprintf("matchmaking-server_message-consumer-%s", consumer)
	return transport.NewRedisMessageGroupConsumer(ctx, rdb, stream, consumerGroup, consumer)
}
