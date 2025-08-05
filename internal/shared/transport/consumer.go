package transport

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils"
	"github.com/redis/go-redis/v9"
)

type WrappedConsumeMsg struct {
	ID      string
	Payload []byte
	AckFunc func() error
}

type MessageConsumer interface {
	StartConsuming(ctx context.Context) (<-chan WrappedConsumeMsg, <-chan error)
	AckMessage(ctx context.Context, msgId string) error
}

type MessageConsumerFactory interface {
	CreateConsumer(ctx context.Context, consumer string) (MessageConsumer, error)
}

type RedisMessageConsumer struct {
	rdb           *redis.Client
	stream        string
	consumerGroup string
	consumer      string
	luaScripts    map[string]*redis.Script
}

var (
	luaScriptBasePath     = "../../../db/redis/scripts"
	atomicAckDelFilePath  = fmt.Sprintf("%s/cgroup_xack_xdel_atomic.lua", luaScriptBasePath)
	consumerBlockDuration = time.Second * 5
)

func NewRedisMessageConsumer(ctx context.Context, rdb *redis.Client, stream, consumerGroup, consumer string) (*RedisMessageConsumer, error) {
	var r *RedisMessageConsumer
	luaScripts := make(map[string]*redis.Script)
	atomicAckDelSrc, err := utils.LoadLuaSrc(atomicAckDelFilePath)
	if err != nil {
		return r, fmt.Errorf("error loading atomicAckDel lua script - %v", err)
	}
	luaScripts[atomicAckDelFilePath] = redis.NewScript(atomicAckDelSrc)

	r = &RedisMessageConsumer{
		rdb:           rdb,
		stream:        stream,
		consumerGroup: consumerGroup,
		consumer:      consumer,
	}
	_, err = r.rdb.XGroupCreateMkStream(ctx, stream, consumerGroup, "$").Result()
	return r, err
}

func (mc *RedisMessageConsumer) StartConsuming(ctx context.Context) (<-chan WrappedConsumeMsg, <-chan error) {
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
				data, ok := streamResults[0].Messages[0].Values["payload"].(string)
				if !ok {
					fmt.Printf("MatchmakingClientMessageConsumer has error trying to structure the stream reply - %v", err)
				}
				bytes := []byte(data)
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

func (mc *RedisMessageConsumer) AckMessage(ctx context.Context, msgId string) error {
	return mc.luaScripts[atomicAckDelFilePath].Run(ctx, mc.rdb, []string{mc.stream}, mc.consumerGroup, msgId).Err()
}
