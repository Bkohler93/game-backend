package matchmake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bkohler93/game-backend/internal/shared/constants/metadata"
	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

const (
	ServerMessageConsumer         transport.MessageConsumerType        = "ServerMessageConsumer"
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

func NewBus(serverMessageConsumer transport.MessageConsumer, clientMessageProducer transport.DynamicMessageProducer, matchmakeWorkerNotifier transport.BroadcastProducer, matchmakeWorkerNotifyReceiver transport.BroadcastConsumer) *TransportBus {
	b := &TransportBus{
		transportBus: &transport.Bus{},
	}
	b.transportBus.AddMessageConsumer(ServerMessageConsumer, serverMessageConsumer)
	b.transportBus.AddDynamicMessageProducer(ClientMessageProducer, clientMessageProducer)
	b.transportBus.AddBroadcastProducer(MatchmakeWorkerNotifier, matchmakeWorkerNotifier)
	b.transportBus.AddBroadcastConsumer(MatchmakeWorkerNotifyReceiver, matchmakeWorkerNotifyReceiver)
	return b
}

func (b *TransportBus) ListenForMatchmakeWorkerNotifications(ctx context.Context) (<-chan *message.Envelope, <-chan error) {
	dataCh, errCh := b.transportBus.Subscribe(ctx, MatchmakeWorkerNotifyReceiver)
	msgCh := make(chan *message.Envelope)
	errorCh := make(chan error)
	go func() {
		for {
			select {
			case msg := <-dataCh:
				msgCh <- msg
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
	//return b.transportBus.Publish(ctx, MatchmakeWorkerNotifier, message.EmptyMessage{})
	return b.transportBus.Publish(ctx, MatchmakeWorkerNotifier, &message.Envelope{
		Type:     "",
		Payload:  nil,
		MetaData: nil,
	})
}

func (b *TransportBus) SendToClient(ctx context.Context, id uuidstring.ID, msg MatchmakingClientMessage) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	var md metadata.MetaData
	if msg.GetDiscriminator() == string(RoomFull) {
		roomFullMsg := msg.(*RoomFullMessage)
		md = make(metadata.MetaData)
		md[metadata.NewGameState] = metadata.Setup
		md[metadata.RoomID] = roomFullMsg.RoomID.String()
	}

	return b.transportBus.SendTo(ctx, ClientMessageProducer, id, &message.Envelope{
		Type:     msg.GetDiscriminator(),
		Payload:  bytes,
		MetaData: md,
	})
}

func (b *TransportBus) StartReceivingServerMessages(ctx context.Context) (<-chan *message.MessageContext, <-chan error) {
	wrappedMsgCh, errCh := b.transportBus.StartReceiving(ctx, ServerMessageConsumer)
	return transport.UnwrapAndForward[MatchmakingServerMessage](ctx, wrappedMsgCh, errCh, serverMessageTypeRegistry)
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
