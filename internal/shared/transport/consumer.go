package transport

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/interfacestruct"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

type MatchmakingClientMessageConsumer interface {
	Consume(ctx context.Context) (message.BaseMatchmakingClientMessage, error)
}

type MatchmakingClientMessageConsumerFactory interface {
	CreateConsumer(ctx context.Context, userId stringuuid.StringUUID) (MatchmakingClientMessageConsumer, error)
}

type RedisMatchmakingClientMessageConsumerFactory struct {
	rdb *redis.Client
}

func NewRedisMatchmakingClientMessageConsumerFactory(rdb *redis.Client) *RedisMatchmakingClientMessageConsumerFactory {
	return &RedisMatchmakingClientMessageConsumerFactory{rdb: rdb}
}

func (r *RedisMatchmakingClientMessageConsumerFactory) CreateConsumer(ctx context.Context, userId stringuuid.StringUUID) (MatchmakingClientMessageConsumer, error) {
	return NewRedisMatchmakingClientMessageConsumer(ctx, r.rdb, userId)
}

// RedisMatchmakingClientMessageConsumer
// - this is created per websocket connection by the Websocket Gateway.
type RedisMatchmakingClientMessageConsumer struct {
	rdb           *redis.Client
	stream        string
	consumerGroup string
	userId        stringuuid.StringUUID
}

func NewRedisMatchmakingClientMessageConsumer(ctx context.Context, rdb *redis.Client, userId stringuuid.StringUUID) (*RedisMatchmakingClientMessageConsumer, error) {
	stream := rediskeys.MatchmakingClientMessageStream(userId)
	cgroup := rediskeys.MatchmakingClientMessageCGroup(userId)
	r := &RedisMatchmakingClientMessageConsumer{
		rdb:           rdb,
		stream:        stream,
		consumerGroup: cgroup,
		userId:        userId,
	}
	_, err := r.rdb.XGroupCreateMkStream(ctx, stream, cgroup, "$").Result()
	return r, err
}

func (r *RedisMatchmakingClientMessageConsumer) Consume(ctx context.Context) (message.BaseMatchmakingClientMessage, error) {
	var msg message.BaseMatchmakingClientMessage

	streamResults, err := r.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.consumerGroup,
		Consumer: r.userId.String(),
		Streams:  []string{r.stream, ">"},
		Count:    1,
	}).Result()
	if err != nil {
		return msg, err
	}
	data := streamResults[0].Messages[0].Values
	err = interfacestruct.Structify(data, &msg)
	return msg, err
}
