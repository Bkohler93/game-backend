package game

import "github.com/bkohler93/game-backend/internal/shared/transport"

// const (
// 	ServerMessageConsumer transport.MessageConsumerType        = "ServerMessageConsumer"
// 	ClientMessageProducer transport.DynamicMessageProducerType = "ClientMessageProducer"
// )

type TransportFactory struct {
	GameServerMsgConsumerBuilder transport.MessageConsumerBuilderFunc
	GameClientMsgProducerBuilder transport.DynamicMessageProducerBuilderFunc
}
