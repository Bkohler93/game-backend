package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/app/gateway"
	"github.com/bkohler93/game-backend/internal/app/matchmake"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

func TestRedisMatchmakingClientMessageConsumerFactory(t *testing.T) {
	ctx := t.Context()

	rdb, err := redisutils.NewRedisClient(ctx)
	if err != nil {
		t.Errorf("unexepcted error when creating redis client - %v", err)
	}
	factory := gateway.NewRedisMatchmakingClientMessageConsumerFactory(rdb)
	userId := stringuuid.NewStringUUID()
	_, err = factory.CreateConsumer(ctx, userId.String())
	if err != nil {
		t.Errorf("unexpected error when creating redisMatchmakingClientMessageConsumer - %v", err)
	}
}

func createFlushFunc(cancel context.CancelFunc, wg *sync.WaitGroup, rdb *redis.Client) func() {
	return func() {
		cancel()

		wg.Wait()

		err := rdb.FlushDB(context.Background()).Err()
		if err != nil {
			panic(err)
		}
	}
}

func TestRedisMatchmakingClientMessageConsumerAndProducer(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var userId stringuuid.StringUUID
	startup := func(t *testing.T) (consumer transport.MessageConsumer, producer transport.DynamicMessageProducer, flush func()) {
		client, err := redisutils.NewRedisClient(ctx)
		if err != nil {
			panic(err)
		}
		userId = stringuuid.NewStringUUID()
		factory := gateway.NewRedisMatchmakingClientMessageConsumerFactory(client)
		consumer, err = factory.CreateConsumer(ctx, userId.String())
		if err != nil {
			panic(err)
		}
		producer = matchmake.NewMatchmakingClientMessageRedisProducer(client)
		flush = func() {
			cancel()
			fmt.Println("context cancelled", ctx.Err())
			err := client.FlushDB(context.Background()).Err()
			if err != nil {
				panic(err)
			}
			fmt.Println("redis has been fluuushed")
		}
		return
	}

	t.Run("producer sends messages and consumer receives them", func(t *testing.T) {
		consumer, producer, flush := startup(t)
		defer flush()

		var userIds []stringuuid.StringUUID
		messageCount := 4
		for i := 0; i < messageCount; i++ {
			newId := stringuuid.NewStringUUID()
			userIds = append(userIds, newId)

			payload := matchmake.NewPlayerJoinedRoomMessage(newId)
			fmt.Println("sending--", payload)
			bytes, err := json.Marshal(payload)
			if err != nil {
				t.Errorf("failed to marshal msg - %v", err)
			}
			fmt.Println("bytes being sent --- ", bytes)

			err = producer.PublishTo(ctx, userId, bytes)
			if err != nil {
				t.Errorf("unexpected error trying to publish msg - %v", err)
			}
		}
		doneCh := make(chan int)
		errCh := make(chan error)
		ticker := time.NewTicker(time.Second * 2)

		go func() {
			matchmakingClientMsgCh, matchmakingMsgErrCh := consumer.StartConsuming(ctx)
			count := 0
			for {
				select {
				case <-ticker.C:
					ticker.Stop()
					errCh <- errors.New("timed out while waiting for consumer to receive message")
					return
				case msgErr := <-matchmakingMsgErrCh:
					t.Errorf("error while consuming from MatchmakingClientMessage stream - %v", msgErr)
				case msg := <-matchmakingClientMsgCh:
					payload := msg.Payload
					mmMsg, err := matchmake.UnmarshalMatchmakingClientMessage(payload)
					if err != nil {
						t.Errorf("encountered error trying to unmarshal msg recieved by consumer - %v", err)
					}
					playerJoinedRoomMsg, ok := mmMsg.(*matchmake.PlayerJoinedRoomMessage)
					if !ok {
						t.Errorf("expected a PlayerJoinedRoomMessage, got %v", reflect.TypeOf(mmMsg))
					}
					if !slices.Contains(userIds, playerJoinedRoomMsg.UserJoinedId) {
						t.Errorf("consumer received unknown player id %s", playerJoinedRoomMsg.UserJoinedId)
					}
					count++
					if count == messageCount {
						ticker.Stop()
						doneCh <- 1
						return
					}
				}
			}
		}()

		select {
		case <-doneCh:
			break
		case err := <-errCh:
			t.Error(err)
		}
	})
}
