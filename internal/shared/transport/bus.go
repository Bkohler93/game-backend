package transport

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type Bus struct {
	//messageGroupConsumers map[MessageGroupConsumerType]MessageGroupConsumer
	broadcastConsumers map[BroadcastConsumerType]BroadcastConsumer
	messageConsumers   map[MessageConsumerType]MessageConsumer

	dynamicMessageProducers map[DynamicMessageProducerType]DynamicMessageProducer
	messageProducers        map[MessageProducerType]MessageProducer
	broadcastProducers      map[BroadcastProducerType]BroadcastProducer
}

func genericAdd[K comparable, V any](m *map[K]V, key K, value V) {
	if *m == nil {
		*m = make(map[K]V)
	}
	(*m)[key] = value
}

//func (m *Bus) AddMessageGroupConsumer(t MessageGroupConsumerType, consumer MessageGroupConsumer) {
//	genericAdd(&m.messageGroupConsumers, t, consumer)
//}

func (m *Bus) AddBroadcastConsumer(t BroadcastConsumerType, consumer BroadcastConsumer) {
	genericAdd(&m.broadcastConsumers, t, consumer)
}

func (m *Bus) AddMessageConsumer(t MessageConsumerType, consumer MessageConsumer) {
	genericAdd(&m.messageConsumers, t, consumer)
}

func (m *Bus) AddDynamicMessageProducer(t DynamicMessageProducerType, producer DynamicMessageProducer) {
	genericAdd(&m.dynamicMessageProducers, t, producer)
}

func (m *Bus) AddMessageProducer(t MessageProducerType, producer MessageProducer) {
	genericAdd(&m.messageProducers, t, producer)
}

func (m *Bus) AddBroadcastProducer(t BroadcastProducerType, producer BroadcastProducer) {
	genericAdd(&m.broadcastProducers, t, producer)
}

func (m *Bus) Send(ctx context.Context, t MessageProducerType, env *message.Envelope) error {
	return m.messageProducers[t].Send(ctx, env)
}

func (m *Bus) SendTo(ctx context.Context, producerType DynamicMessageProducerType, recipient uuidstring.ID, env *message.Envelope) error {
	return m.dynamicMessageProducers[producerType].SendTo(ctx, recipient, env)
}

func (m *Bus) Publish(ctx context.Context, producerType BroadcastProducerType, env *message.Envelope) error {
	return m.broadcastProducers[producerType].Publish(ctx, env)
}

func (m *Bus) StartReceiving(ctx context.Context, consumerType MessageConsumerType) (msgCh <-chan *message.EnvelopeContext, errCh <-chan error) {
	return m.messageConsumers[consumerType].StartReceiving(ctx)
}

//	func (m *Bus) AckMessage(ctx context.Context, consumerType MessageGroupConsumerType, msgId string) error {
//		return m.messageGroupConsumers[consumerType].AckMessage(ctx, msgId)
//	}
func (m *Bus) Subscribe(ctx context.Context, consumerType BroadcastConsumerType) (<-chan *message.Envelope, <-chan error) {
	return m.broadcastConsumers[consumerType].Subscribe(ctx)
}
