package transport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

type MessageProducerType string
type MessageProducer interface {
	Send(ctx context.Context, msg any) error
}
type MessageProducerBuilderFunc = func() MessageProducer

type DynamicMessageProducerType string
type DynamicMessageProducer interface {
	SendTo(ctx context.Context, recipientId uuidstring.ID, msg any) error
}

type BroadcastProducerType string
type BroadcastProducer interface {
	Publish(ctx context.Context, msg any) error
}

type RedisDynamicMessageProducer struct {
	rdb    *redis.Client
	stream func(uuidstring.ID) string
}

func (r *RedisDynamicMessageProducer) SendTo(ctx context.Context, recipientId uuidstring.ID, msg any) error {
	fmt.Println("object to marshal: ", msg)
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"payload": data,
	}
	fmt.Println("values", values)
	stream := r.stream(recipientId)
	fmt.Println("publishing to: ", stream)
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

func (r *RedisMessageProducer) Send(ctx context.Context, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
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
