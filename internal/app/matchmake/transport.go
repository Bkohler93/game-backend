package matchmake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

const (
	ServerMessageConsumer         transport.MessageGroupConsumerType   = "ServerMessageConsumer"
	ClientMessageProducer         transport.DynamicMessageProducerType = "ClientMessageProducer"
	MatchmakeWorkerNotifier       transport.BroadcastProducerType      = "MatchmakeWorkerNotifier"
	MatchmakeWorkerNotifyReceiver transport.BroadcastConsumerType      = "MatchmakeWorkerNotifyReceiver"
)

var (
	ErrExpectedString = errors.New("expected string")
)

type TransportBus struct {
	transportBus *transport.Bus
}

func NewBus(serverMessageConsumer transport.MessageGroupConsumer, clientMessageProducer transport.DynamicMessageProducer, matchmakeWorkerNotifier transport.BroadcastProducer, matchmakeWorkerNotifyReceiver transport.BroadcastConsumer) *TransportBus {
	b := &TransportBus{
		transportBus: &transport.Bus{},
	}
	b.transportBus.AddMessageGroupConsumer(ServerMessageConsumer, serverMessageConsumer)
	b.transportBus.AddDynamicMessageProducer(ClientMessageProducer, clientMessageProducer)
	b.transportBus.AddBroadcastProducer(MatchmakeWorkerNotifier, matchmakeWorkerNotifier)
	b.transportBus.AddBroadcastConsumer(MatchmakeWorkerNotifyReceiver, matchmakeWorkerNotifyReceiver)
	return b
}

func (b *TransportBus) ListenForMatchmakeWorkerNotifications(ctx context.Context) (<-chan string, <-chan error) {
	dataCh, errCh := b.transportBus.Subscribe(ctx, MatchmakeWorkerNotifyReceiver)
	msgCh := make(chan string)
	errorCh := make(chan error)
	go func() {
		for {
			select {
			case msg := <-dataCh:
				str, ok := msg.(string)
				if !ok {
					errorCh <- ErrExpectedString
					continue
				}
				msgCh <- str
			case <-ctx.Done():
				errorCh <- ctx.Err()
			case err := <-errCh:
				errorCh <- err
			}
		}
	}()
	return msgCh, errorCh
}

func (b *TransportBus) NotifyMatchmakeWorkers(ctx context.Context) error {
	return b.transportBus.Publish(ctx, MatchmakeWorkerNotifier, message.EmptyMessage{})
}

func (b *TransportBus) SendToClient(ctx context.Context, id uuidstring.ID, msg MatchmakingClientMessage) error {
	return b.transportBus.SendTo(ctx, ClientMessageProducer, id, msg, msg.RemoveMetaData())
}

func (b *TransportBus) StartReceivingServerMessages(ctx context.Context) (<-chan MatchmakingServerMessage, <-chan error) {
	wrappedMsgCh, errCh := b.transportBus.StartReceiving(ctx, ServerMessageConsumer)
	return transport.UnwrapAndForward[MatchmakingServerMessage](ctx, wrappedMsgCh, errCh, serverMessageTypeRegistry)
}

func (b *TransportBus) AckServerMessage(ctx context.Context, id string) error {
	return b.transportBus.AckMessage(ctx, ServerMessageConsumer, id)
}

func NewRedisMatchmakingServerMessageConsumer(ctx context.Context, rdb *redis.Client, consumer string) (*transport.RedisMessageGroupConsumer, error) {
	stream := rediskeys.MatchmakingServerMessageStream
	consumerGroup := rediskeys.MatchmakingServerMessageCGroup
	consumer = fmt.Sprintf("matchmaking-server_message-consumer-%s", consumer)
	return transport.NewRedisMessageGroupConsumer(ctx, rdb, stream, consumerGroup, consumer)
}

func NewRedisClientMessageProducer(rdb *redis.Client) *transport.RedisDynamicMessageProducer {
	return transport.NewRedisDynamicMessageProducer(rdb, rediskeys.MatchmakingClientMessageStream)
}

func NewRedisWorkerNotifierBroadcastProducer(rdb *redis.Client) *transport.RedisBroadcastProducer {
	return transport.NewRedisBroadcastProducer(rdb, rediskeys.MatchmakeNotifyWorkersPubSub)
}

func NewRedisWorkerNotifierListener(rdb *redis.Client) *transport.RedisBroadcastConsumer {
	return transport.NewRedisBroadcastConsumer(rdb, rediskeys.MatchmakeNotifyWorkersPubSub)
}
