package integration

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/app/gateway"
	"github.com/bkohler93/game-backend/internal/app/matchmake"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

func TestRedisMatchmakingClientMessageConsumerFactory(t *testing.T) {
	ctx := t.Context()

	rdb, err := redisutils.NewRedisMatchmakeClient(ctx)
	if err != nil {
		t.Errorf("unexepcted error when creating redis client - %v", err)
	}
	factory := gateway.NewRedisMatchmakingClientMessageConsumerFactory(rdb)
	userId := uuidstring.NewID()
	_, err = factory.CreateGroupConsumer(ctx, userId.String())
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
	var userId uuidstring.ID
	startup := func(t *testing.T) (consumer transport.MessageGroupConsumer, producer transport.DynamicMessageProducer, flush func()) {
		client, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		userId = uuidstring.NewID()
		factory := gateway.NewRedisMatchmakingClientMessageConsumerFactory(client)
		consumer, err = factory.CreateGroupConsumer(ctx, userId.String())
		if err != nil {
			panic(err)
		}
		producer = transport.NewRedisDynamicMessageProducer(client, rediskeys.MatchmakingClientMessageStream)
		flush = func() {
			cancel()
			err := client.FlushDB(context.Background()).Err()
			if err != nil {
				panic(err)
			}
		}
		return
	}

	t.Run("producer sends messages and consumer receives them", func(t *testing.T) {
		consumer, producer, flush := startup(t)
		defer flush()

		var userIds []uuidstring.ID
		messageCount := 4
		for i := 0; i < messageCount; i++ {
			newId := uuidstring.NewID()
			userIds = append(userIds, newId)

			payload := matchmake.NewPlayerJoinedRoomMessage(newId)
			bytes, err := json.Marshal(payload)
			if err != nil {
				t.Errorf("error marshalling message - %v", err)
			}

			err = producer.SendTo(ctx, userId, bytes)
			if err != nil {
				t.Errorf("unexpected error trying to publish msg - %v", err)
			}
		}
		doneCh := make(chan int)
		errCh := make(chan error)
		ticker := time.NewTicker(time.Second * 2)

		go func() {
			matchmakingClientMsgCh, matchmakingMsgErrCh := consumer.StartReceiving(ctx)
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
					payload := msg.Payload.([]byte)
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

func TestRedisMatchmakingServerMessageConsumerAndProducer(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var serverId uuidstring.ID
	startup := func(t *testing.T) (consumer transport.MessageGroupConsumer, producer transport.MessageProducer, flush func()) {
		client, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		serverId = uuidstring.NewID()
		consumer, err = matchmake.NewRedisMatchmakingServerMessageConsumer(ctx, client, string(serverId))
		if err != nil {
			panic(err)
		}
		producer = transport.NewRedisMessageProducer(client, rediskeys.MatchmakingServerMessageStream)
		flush = func() {
			cancel()
			err := client.FlushDB(context.Background()).Err()
			if err != nil {
				panic(err)
			}
		}
		return
	}

	t.Run("producer sends messages and consumer receives them", func(t *testing.T) {
		consumer, producer, flush := startup(t)
		defer flush()

		name := "ButtholeSmeller"
		timeCreated := time.Now().Unix()
		skill := 100
		region := "na"
		messageCount := 4
		newId := uuidstring.NewID()
		originalMsg := matchmake.NewRequestMatchmakingMessage(newId, name, timeCreated, skill, region)
		for i := 0; i < messageCount; i++ {

			payload := originalMsg

			bytes, err := json.Marshal(payload)
			if err != nil {
				t.Errorf("error trying to marshal payload - %v", err)
			}

			err = producer.Send(ctx, bytes)
			if err != nil {
				t.Errorf("unexpected error trying to publish msg - %v", err)
			}
		}
		doneCh := make(chan int)
		errCh := make(chan error)
		ticker := time.NewTicker(time.Second * 2)

		go func() {
			matchmakingServerMsgCh, matchmakingServerErrCh := consumer.StartReceiving(ctx)
			count := 0
			for {
				select {
				case <-ticker.C:
					ticker.Stop()
					errCh <- errors.New("timed out while waiting for consumer to receive message")
					return
				case msgErr := <-matchmakingServerErrCh:
					t.Errorf("error while consuming from MatchmakingClientMessage stream - %v", msgErr)
				case msg := <-matchmakingServerMsgCh:
					payload := msg.Payload
					mmMsg, err := matchmake.UnmarshalMatchmakingServerMessage(payload.([]byte))
					if err != nil {
						t.Errorf("encountered error trying to unmarshal msg recieved by consumer - %v", err)
					}
					switch msg := mmMsg.(type) {
					case *matchmake.ExitMatchmakingMessage:
						t.Errorf("did not expect exit matchmaking message - %v", msg)
					case *matchmake.RequestMatchmakingMessage:
						if !msg.Equals(originalMsg) {
							t.Errorf("expected consumed message {%v} to equal original message, got - {%v}", msg, originalMsg)
						}
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
