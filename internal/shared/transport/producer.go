package transport

import (
	"context"
	"encoding/json"

	"github.com/bkohler93/game-backend/internal/shared/constants/metadata"
	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

type MessageProducerType string
type MessageProducer interface {
	Send(ctx context.Context, data message.Message, metaData metadata.MetaData) error
}
type MessageProducerBuilderFunc = func() MessageProducer

type DynamicMessageProducerType string
type DynamicMessageProducer interface {
	SendTo(ctx context.Context, recipientId uuidstring.ID, data message.Message, metaData metadata.MetaData) error
}
type DynamicMessageProducerBuilderFunc = func() DynamicMessageProducer

type BroadcastProducerType string
type BroadcastProducer interface {
	Publish(ctx context.Context, data message.Message) error
}

type RedisDynamicMessageProducer struct {
	rdb    *redis.Client
	stream func(uuidstring.ID) string
}

func (r *RedisDynamicMessageProducer) SendTo(ctx context.Context, recipientId uuidstring.ID, data message.Message, metaData metadata.MetaData) error {
	values := map[string]interface{}{
		"payload":            data,
		metadata.MetaDataKey: metaData,
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

func (r *RedisMessageProducer) Send(ctx context.Context, msg message.Message, metaData metadata.MetaData) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	md, err := json.Marshal(metaData)
	if err != nil {
		return err
	}

	values := map[string]interface{}{
		"payload":            payload,
		metadata.MetaDataKey: md,
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

type RedisBroadcastProducer struct {
	rdb     *redis.Client
	channel string
}

func NewRedisBroadcastProducer(rdb *redis.Client, channel string) *RedisBroadcastProducer {
	return &RedisBroadcastProducer{rdb, channel}
}

func (r *RedisBroadcastProducer) Publish(ctx context.Context, msg message.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = r.rdb.Publish(ctx, r.channel, payload).Err()
	return err
}
