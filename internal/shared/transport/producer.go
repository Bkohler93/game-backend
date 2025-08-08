package transport

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

type MessageProducerType string
type MessageProducer interface {
	Send(ctx context.Context, data []byte) error
}
type MessageProducerBuilderFunc = func() MessageProducer

type DynamicMessageProducerType string
type DynamicMessageProducer interface {
	SendTo(ctx context.Context, recipientId uuidstring.ID, data []byte) error
}

type BroadcastProducerType string
type BroadcastProducer interface {
	Publish(ctx context.Context, msg any) error
}

type RedisDynamicMessageProducer struct {
	rdb    *redis.Client
	stream func(uuidstring.ID) string
}

func (r *RedisDynamicMessageProducer) SendTo(ctx context.Context, recipientId uuidstring.ID, data []byte) error {
	values := map[string]interface{}{
		"payload": data,
	}
	stream := r.stream(recipientId)
	return r.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		ID:     "*",
		Values: values,
	}).Err()
}

func NewRedisDynamicMessageProducer(rdb *redis.Client, streamNameFunc func(uuidstring.ID) string) *RedisDynamicMessageProducer {
	return &RedisDynamicMessageProducer{
		rdb:    rdb,
		stream: streamNameFunc,
	}
}

type RedisMessageProducer struct {
	rdb    *redis.Client
	stream string
}

func (r *RedisMessageProducer) Send(ctx context.Context, data []byte) error {
	//data, err := json.Marshal(msg)
	//if err != nil {
	//	return err
	//}
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
