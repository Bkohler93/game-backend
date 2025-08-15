package gateway

import (
	"context"
	"encoding/json"

	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

type Router struct {
	busFactory *TransportBusFactory
}

func (r *Router) RouteClientTraffic(ctx context.Context, client *Client) {
	bus := r.busFactory.NewTransportBus(client.ID)
	eg, eCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		msgCh, errCh := bus.StartReceivingMatchmakingClientMessages(eCtx)
		for {
			select {
			case msg := <-msgCh:
				//TODO msg.Metadata[""] <- use MetaData to figure out if we need to start Receiving Game/Setup messages

				// check if RoomFull msg, then start receiving from GameSetupMessages/GamePlayMessages
				client.outChan <- msg
			}
		}
	})
}

func (r *Router) TeardownClientRouting(client *Client) {

}

const (
	ClientMessageConsumer transport.MessageGroupConsumerType = "ClientMessageConsumer"
	ServerMessageProducer transport.MessageProducerType      = "ServerMessageProducer"
)

type TransportBusFactory struct {
	rdb                                 *redis.Client
	matchmakingClientMsgConsumerBuilder transport.MessageGroupConsumerBuilderFunc
	matchmakingServerMsgProducerBuilder transport.MessageProducerBuilderFunc
}

func NewClientTransportBusFactory(rdb *redis.Client, matchmakingClientMsgConsumerBuilder transport.MessageGroupConsumerBuilderFunc, matchmakingServerMessageProducerBuilder transport.MessageProducerBuilderFunc) *TransportBusFactory {
	return &TransportBusFactory{
		rdb,
		matchmakingClientMsgConsumerBuilder,
		matchmakingServerMessageProducerBuilder,
	}
}

type TransportBus struct {
	bus *transport.Bus
}

func (f *TransportBusFactory) NewTransportBus(clientId uuidstring.ID) *TransportBus {
	b := &TransportBus{
		bus: &transport.Bus{},
	}
	clientMessageConsumer := f.matchmakingClientMsgConsumerBuilder(clientId.String())
	serverMessageProducer := f.matchmakingServerMsgProducerBuilder()

	b.bus.AddMessageGroupConsumer(ClientMessageConsumer, clientMessageConsumer)
	b.bus.AddMessageProducer(ServerMessageProducer, serverMessageProducer)
	return b
}

func (b *TransportBus) StartReceivingMatchmakingClientMessages(ctx context.Context) (<-chan transport.AckableMessage, <-chan error) {
	return b.bus.StartReceiving(ctx, ClientMessageConsumer)
}

func (b *TransportBus) AckMatchmakingMsg(ctx context.Context, id string) error {
	return b.bus.AckMessage(ctx, ClientMessageConsumer, id)
}

func (b *TransportBus) SendMatchmakingServerMessage(ctx context.Context, payload json.RawMessage) error {
	return b.bus.Send(ctx, ServerMessageProducer, payload)
}
