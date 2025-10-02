package gateway

import "github.com/bkohler93/game-backend/internal/shared/transport"

// const (
// 	ClientMessageConsumer transport.MessageConsumerType = "ClientMessageConsumer"
// 	ServerMessageProducer transport.MessageProducerType = "ServerMessageProducer"
// )

type TransportFactory struct {
	MatchmakingClientMsgConsumerBuilder transport.MessageConsumerBuilderFunc
	GameClientMsgConsumerBuilder        transport.MessageConsumerBuilderFunc

	MatchmakingServerMsgProducerBuilder transport.MessageProducerBuilderFunc
	GameplayServerMsgProducerBuilder    transport.DynamicMessageProducerBuilderFunc
}
