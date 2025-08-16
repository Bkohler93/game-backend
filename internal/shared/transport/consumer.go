package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/constants/metadata"
	"github.com/bkohler93/game-backend/internal/shared/utils/files"
	"github.com/redis/go-redis/v9"
)

type TransportMessage interface {
	Ack(context.Context) error
	GetPayload() any
	GetMetadata() map[string]interface{}
}

type AckableMessage struct {
	AckFunc  func(context.Context) error
	Payload  any
	Metadata map[string]interface{}
}

func (a AckableMessage) Ack(ctx context.Context) error {
	if a.AckFunc != nil {
		return a.AckFunc(ctx)
	}
	return nil
}

func (a AckableMessage) GetPayload() any {
	return a.Payload
}

func (a AckableMessage) GetMetadata() map[string]interface{} {
	return a.Metadata
}

type Message struct {
	Payload  any
	Metadata map[string]interface{}
}

func (m Message) Ack(ctx context.Context) error {
	return nil
}

func (m Message) GetPayload() any {
	return m.Payload
}

func (m Message) GetMetadata() map[string]interface{} {
	return m.Metadata
}

var (
	ErrScriptNotFound = errors.New("script not found")
)

type MessageGroupConsumerType string
type MessageGroupConsumer interface {
	StartReceiving(ctx context.Context) (<-chan AckableMessage, <-chan error)
	AckMessage(ctx context.Context, msgId string) error
}

type MessageConsumerType string
type MessageConsumer interface {
	StartReceiving(ctx context.Context) (<-chan Message, <-chan error)
}
type MessageConsumerBuilderFunc func(streamSuffix string) MessageConsumer

type RedisMessageConsumer struct {
	rdb            *redis.Client
	stream         string
	lastReceivedID string
}

func NewRedisMessageConsumer(rdb *redis.Client, stream string) *RedisMessageConsumer {
	return &RedisMessageConsumer{
		rdb:            rdb,
		stream:         stream,
		lastReceivedID: "$",
	}
}

func (r RedisMessageConsumer) StartReceiving(ctx context.Context) (<-chan Message, <-chan error) {
	msgCh := make(chan Message)
	errCh := make(chan error, 1)
	go func() {
		defer close(msgCh)
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				streamResults, err := r.rdb.XRead(ctx, &redis.XReadArgs{
					Streams: []string{r.stream},
					Count:   1,
					Block:   0,
					ID:      r.lastReceivedID,
				}).Result()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					errCh <- fmt.Errorf("error reading from RedisMessageConsumer stream - %v", err)
					return
				}
				data, ok := streamResults[0].Messages[0].Values["payload"].(string)
				if !ok {
					log.Printf("error trying to structure the stream reply - %v", err)
				}
				payload := []byte(data)
				r.lastReceivedID = streamResults[0].Messages[0].ID

				var metaData map[string]interface{}
				metaDataString, ok := streamResults[0].Messages[0].Values[metadata.MetaDataKey].(string)
				if ok {
					metaDataBytes := []byte(metaDataString)
					err = json.Unmarshal(metaDataBytes, &metaData)
					if err != nil {
						log.Printf("error trying to unmarshal metadata - %v", err)
					}
				}

				msgCh <- Message{
					Payload:  payload,
					Metadata: metaData,
				}
			}
		}
	}()
	return msgCh, errCh
}

type MessageGroupConsumerBuilderFunc = func(consumerId string) MessageGroupConsumer
type MessageGroupConsumerFactory interface {
	CreateGroupConsumer(ctx context.Context, consumer string) (MessageGroupConsumer, error)
}

type BroadcastConsumerType string
type BroadcastConsumer interface {
	Subscribe(ctx context.Context) (<-chan any, <-chan error)
}

type RedisMessageGroupConsumer struct {
	rdb           *redis.Client
	stream        string
	consumerGroup string
	consumer      string
	luaScripts    map[string]*redis.Script
}

var (
	consumerBlockDuration = time.Second * 5
)

func NewRedisMessageGroupConsumer(ctx context.Context, rdb *redis.Client, stream, consumerGroup, consumer string) (*RedisMessageGroupConsumer, error) {
	var r *RedisMessageGroupConsumer
	luaScripts := make(map[string]*redis.Script)
	atomicAckDelSrc, err := files.GetLuaScript(files.LuaCGroupAckDelMsg)
	if err != nil {
		return r, fmt.Errorf("error loading atomicAckDel lua script - %v", err)
	}
	luaScripts[files.LuaCGroupAckDelMsg] = redis.NewScript(atomicAckDelSrc)

	r = &RedisMessageGroupConsumer{
		rdb:           rdb,
		stream:        stream,
		consumerGroup: consumerGroup,
		consumer:      consumer,
		luaScripts:    luaScripts,
	}
	_, err = r.rdb.XGroupCreateMkStream(ctx, stream, consumerGroup, "$").Result()
	return r, err
}

func (mc *RedisMessageGroupConsumer) StartReceiving(ctx context.Context) (<-chan AckableMessage, <-chan error) {
	msgCh := make(chan AckableMessage)
	errCh := make(chan error, 1)
	go func() {
		defer close(msgCh)
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				streamResults, err := mc.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    mc.consumerGroup,
					Consumer: mc.consumer,
					Streams:  []string{mc.stream, ">"},
					Count:    1,
					Block:    0,
				}).Result()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					errCh <- fmt.Errorf("error reading from RedisMessageGroupConsumer stream - %v", err)
					return
				}
				data, ok := streamResults[0].Messages[0].Values["payload"].(string)
				if !ok {
					log.Printf("error trying to structure the stream reply - %v", err)
				}
				payload := []byte(data)
				id := streamResults[0].Messages[0].ID

				var metaData map[string]interface{}
				metaDataString, ok := streamResults[0].Messages[0].Values[metadata.MetaDataKey].(string)
				if ok {
					metaDataBytes := []byte(metaDataString)
					err = json.Unmarshal(metaDataBytes, &metaData)
					if err != nil {
						log.Printf("error trying to unmarshal metadata - %v", err)
					}
				}

				msgCh <- AckableMessage{
					AckFunc: func(ackCtx context.Context) error {
						return mc.AckMessage(ackCtx, id)
					},
					//ID:      id,
					Payload:  payload,
					Metadata: metaData,
				}
			}
		}
	}()
	return msgCh, errCh
}

func (mc *RedisMessageGroupConsumer) AckMessage(ctx context.Context, msgId string) error {
	if _, ok := mc.luaScripts[files.LuaCGroupAckDelMsg]; !ok {
		return ErrScriptNotFound
	}
	return mc.luaScripts[files.LuaCGroupAckDelMsg].Run(ctx, mc.rdb, []string{mc.stream}, mc.consumerGroup, msgId).Err()
}

type RedisBroadcastConsumer struct {
	rdb     *redis.Client
	channel string
}

func (r *RedisBroadcastConsumer) Subscribe(ctx context.Context) (<-chan any, <-chan error) {
	returnCh := make(chan any)
	errCh := make(chan error)

	receiveCh := r.rdb.Subscribe(ctx, r.channel).Channel()
	go func() {
		for {
			select {
			case msg := <-receiveCh:
				payload := msg.Payload
				returnCh <- payload
			case <-ctx.Done():
				errCh <- ctx.Err()
			}
		}
	}()

	return returnCh, errCh
}

func NewRedisBroadcastConsumer(rdb *redis.Client, channel string) *RedisBroadcastConsumer {
	r := &RedisBroadcastConsumer{rdb, channel}
	return r
}
