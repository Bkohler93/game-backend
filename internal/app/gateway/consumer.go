package gateway

// type RedisMatchmakingClientMessageConsumerFactory struct {
// 	rdb *redis.Client
// }

// func NewRedisMatchmakingClientMessageConsumerFactory(rdb *redis.Client) *RedisMatchmakingClientMessageConsumerFactory {
// 	return &RedisMatchmakingClientMessageConsumerFactory{rdb}
// }

// func (r *RedisMatchmakingClientMessageConsumerFactory) CreateGroupConsumer(ctx context.Context, consumer string) (transport.MessageConsumer, error) {
// 	stream := rediskeys.MatchmakingClientMessageStream(uuidstring.ID(consumer))
// 	consumerGroup := rediskeys.MatchmakingClientMessageCGroup(uuidstring.ID(consumer))
// 	return transport.NewRedisMessageGroupConsumer(ctx, r.rdb, stream, consumerGroup, consumer)
// }
