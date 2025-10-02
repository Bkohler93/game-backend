package game

import (
	"context"
	"encoding/json"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/message/metadata"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	ServerMessageConsumer transport.MessageConsumerType        = "ServerMessageConsumer"
	ClientMessageProducer transport.DynamicMessageProducerType = "ClientMessageProducer"
)

type TransportFactory struct {
	GameServerMsgConsumerBuilder transport.MessageConsumerBuilderFunc
	GameClientMsgProducerBuilder transport.DynamicMessageProducerBuilderFunc
}

type GameTransportBus struct {
	transportBus *transport.Bus
}

func NewGameTransportBus(serverMessageConsumer transport.MessageConsumer, clientMessageProducer transport.DynamicMessageProducer) *GameTransportBus {
	b := &GameTransportBus{
		transportBus: &transport.Bus{},
	}
	b.transportBus.AddMessageConsumer(ServerMessageConsumer, serverMessageConsumer)
	b.transportBus.AddDynamicMessageProducer(ClientMessageProducer, clientMessageProducer)
	return b
}

func (b *GameTransportBus) SendToClient(ctx context.Context, id uuidstring.ID, msg GameClientMessage) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	var md metadata.MetaData

	return b.transportBus.SendTo(ctx, ClientMessageProducer, id, &message.Envelope{
		Type:     message.GameService,
		Payload:  bytes,
		MetaData: md,
	})
}

func (b *GameTransportBus) StartReceivingServerMessages(ctx context.Context) (<-chan *message.MessageContext, <-chan error) {
	wrappedMsgCh, errCh := b.transportBus.StartReceiving(ctx, ServerMessageConsumer)
	return transport.UnwrapAndForward(ctx, wrappedMsgCh, errCh, serverMessageTypeRegistry)
}
