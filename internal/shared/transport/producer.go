package transport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

type MessageProducer interface {
	Publish(ctx context.Context, data json.RawMessage) error
}

type DynamicMessageProducer interface {
	PublishTo(ctx context.Context, recipientId stringuuid.StringUUID, data json.RawMessage) error
}

type MessageProducerFactory interface {
	CreateProducer(ctx context.Context) (MessageProducer, error)
}

type RedisDynamicMessageProducer struct {
	rdb    *redis.Client
	stream func(stringuuid.StringUUID) string
}

func (r *RedisDynamicMessageProducer) PublishTo(ctx context.Context, recipientId stringuuid.StringUUID, data json.RawMessage) error {
	values := map[string]interface{}{
		"payload": []byte(data),
	}
	stream := r.stream(recipientId)
	fmt.Println("publishing to: ", stream)
	return r.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		ID:     "*",
		Values: values,
	}).Err()
}

func NewRedisDynamicMessageProducer(rdb *redis.Client, streamNameFunc func(stringuuid.StringUUID) string) *RedisDynamicMessageProducer {
	return &RedisDynamicMessageProducer{
		rdb:    rdb,
		stream: streamNameFunc,
	}
}

type RedisMessageProducer struct {
	rdb    *redis.Client
	stream string
}

func (r *RedisMessageProducer) Publish(ctx context.Context, data json.RawMessage) error {
	values := map[string]interface{}{
		"payload": data,
	}

	return r.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: r.stream,
		ID:     "*",
		Values: values,
	}).Err()
}

func NewRedisMessageProducer(rdb *redis.Client, stream string) *RedisMessageProducer {
	return &RedisMessageProducer{
		rdb:    rdb,
		stream: stream,
	}
}
