package transport

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/utils/files"
	"github.com/redis/go-redis/v9"
)

type WrappedConsumeMsg struct {
	ID      string
	Payload any
}

type MessageGroupConsumerType string
type MessageGroupConsumer interface {
	StartReceiving(ctx context.Context) (<-chan WrappedConsumeMsg, <-chan error)
	AckMessage(ctx context.Context, msgId string) error
}

type MessageConsumerType string
type MessageConsumer interface {
	StartReceiving(ctx context.Context) (<-chan WrappedConsumeMsg, <-chan error)
}

type MessageGroupConsumerBuilderFunc = func(consumerId string) MessageGroupConsumer
type MessageGroupConsumerFactory interface {
	CreateGroupConsumer(ctx context.Context, consumer string) (MessageGroupConsumer, error)
}

type BroadcastConsumerType string
type BroadcastConsumer interface {
	Subscribe(ctx context.Context) (<-chan message.Discriminable, error)
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
	}
	_, err = r.rdb.XGroupCreateMkStream(ctx, stream, consumerGroup, "$").Result()
	return r, err
}

func (mc *RedisMessageGroupConsumer) StartReceiving(ctx context.Context) (<-chan WrappedConsumeMsg, <-chan error) {
	msgCh := make(chan WrappedConsumeMsg)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Println("listening for messages on stream ---- ", mc.stream)
				streamResults, err := mc.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    mc.consumerGroup,
					Consumer: mc.consumer,
					Streams:  []string{mc.stream, ">"},
					Count:    1,
					Block:    consumerBlockDuration,
				}).Result()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					errCh <- fmt.Errorf("error reading from MatchmakingClientMessage stream - %v", err)
					return
				}
				fmt.Println("Received values", streamResults[0].Messages[0].Values)
				data, ok := streamResults[0].Messages[0].Values["payload"].(string)
				fmt.Println("received data", data)
				if !ok {
					fmt.Printf("MatchmakingClientMessageConsumer has error trying to structure the stream reply - %v", err)
				}
				bytes := []byte(data)
				fmt.Println("bytes", bytes)
				id := streamResults[0].Messages[0].ID

				msgCh <- WrappedConsumeMsg{
					ID:      id,
					Payload: bytes,
				}
			}
		}
	}()
	return msgCh, errCh
}

func (mc *RedisMessageGroupConsumer) AckMessage(ctx context.Context, msgId string) error {
	return mc.luaScripts[files.LuaCGroupAckDelMsg].Run(ctx, mc.rdb, []string{mc.stream}, mc.consumerGroup, msgId).Err()
}
