package message

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisStreamClient struct {
	rdb *redis.Client
}

func NewRedisStreamClient(redisClient *redis.Client) *RedisStreamClient {
	return &RedisStreamClient{
		rdb: redisClient,
	}
}

func (r *RedisStreamClient) Publish(ctx context.Context, channel string, msg any) error {
	return nil
}

func (r *RedisStreamClient) Consume(ctx context.Context, channel string, output any) error {
	// entries, err := m.rdb.XRead(ctx, &goredis.XReadArgs{
	// 	Streams: []string{"matchmake:request", "$"},
	// 	Count:   1,
	// 	Block:   0,
	// }).Result()
	// if err != nil {
	// 	fmt.Printf("failed to read from matchmake:request stream - %v\n", err)
	// 	continue
	// }
	// res := entries[0].Messages[0].Values
	// var req MatchRequest
	// err = interfacestruct.Structify(res, &req)
	// r.rdb.XReadGroup()
	return nil
}
