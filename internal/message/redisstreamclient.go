package message

import (
	"context"

	"github.com/bkohler93/game-backend/internal/redis"
)

type RedisStreamClient struct {
	rdb *redis.RedisClient
}

func NewRedisStreamClient(redisClient *redis.RedisClient) *RedisStreamClient {
	return &RedisStreamClient{
		rdb: redisClient,
	}
}

func (r *RedisStreamClient) Publish(ctx context.Context, channel string, msg []byte) error {

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
	r.rdb.XReadGroup()
}
