package matchmake

import (
	"context"
	"encoding/json"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	ServerMessageConsumer transport.MessageGroupConsumerType = "ServerMessageConsumer"
)

const (
	ClientMessageProducer transport.DynamicMessageProducerType = "ClientMessageProducer"
)

type TransportBus struct {
	transportBus *transport.Bus
}

func (b *TransportBus) SendToClient(ctx context.Context, id uuidstring.ID, msg MatchmakingClientMessage) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return b.transportBus.SendTo(ctx, ClientMessageProducer, id, bytes)
}

func (b *TransportBus) StartReceivingServerMessages(ctx context.Context) (<-chan MatchmakingServerMessage, <-chan error) {
	wrappedMsgCh, errCh := b.transportBus.StartReceiving(ctx, ServerMessageConsumer)
	return transport.UnwrapAndForward[MatchmakingServerMessage](ctx, wrappedMsgCh, errCh, serverMessageTypeRegistry)
}

func (b *TransportBus) AckServerMessage(ctx context.Context, id string) error {
	return b.transportBus.AckMessage(ctx, ServerMessageConsumer, id)
}

func NewBus(serverMessageConsumer transport.MessageGroupConsumer, clientMessageProducer transport.DynamicMessageProducer) *TransportBus {
	b := &TransportBus{
		transportBus: transport.NewBus(),
	}
	b.transportBus.AddMessageGroupConsumer(ServerMessageConsumer, serverMessageConsumer)
	b.transportBus.AddDynamicMessageProducer(ClientMessageProducer, clientMessageProducer)
	return b
}
