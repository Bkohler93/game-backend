package client

import (
	"context"
	"encoding/json"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

const (
	ClientMessageConsumer transport.MessageGroupConsumerType = "ClientMessageConsumer"
	ServerMessageProducer transport.MessageProducerType      = "ServerMessageProducer"
)

type ClientTransportBusFactory struct {
	rdb                                 *redis.Client
	matchmakingClientMsgConsumerBuilder transport.MessageGroupConsumerBuilderFunc
	matchmakingServerMsgProducerBuilder transport.MessageProducerBuilderFunc
}

func NewClientTransportBusFactory(rdb *redis.Client, matchmakingClientMsgConsumerBuilder transport.MessageGroupConsumerBuilderFunc, matchmakingServerMessageProducerBuilder transport.MessageProducerBuilderFunc) *ClientTransportBusFactory {
	return &ClientTransportBusFactory{
		rdb,
		matchmakingClientMsgConsumerBuilder,
		matchmakingServerMessageProducerBuilder,
	}
}

type ClientTransportBus struct {
	transportBus *transport.Bus
}

func (f *ClientTransportBusFactory) NewClientTransportBus(clientId uuidstring.ID) *ClientTransportBus {
	b := &ClientTransportBus{
		transportBus: transport.NewBus(),
	}
	clientMessageConsumer := f.matchmakingClientMsgConsumerBuilder(clientId.String())
	serverMessageProducer := f.matchmakingServerMsgProducerBuilder()

	b.transportBus.AddMessageGroupConsumer(ClientMessageConsumer, clientMessageConsumer)
	b.transportBus.AddMessageProducer(ServerMessageProducer, serverMessageProducer)
	return b
}

func (b *ClientTransportBus) StartReceivingMatchmakingClientMessages(ctx context.Context) (<-chan transport.WrappedConsumeMsg, <-chan error) {
	return b.transportBus.StartReceiving(ctx, ClientMessageConsumer)
}

func (b *ClientTransportBus) AckMatchmakingMsg(ctx context.Context, id string) error {
	return b.transportBus.AckMessage(ctx, ClientMessageConsumer, id)
}

func (b *ClientTransportBus) SendMatchmakingServerMessage(ctx context.Context, payload json.RawMessage) error {
	return b.transportBus.Send(ctx, ServerMessageProducer, payload)
}
