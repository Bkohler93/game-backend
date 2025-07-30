package redisutils

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient
// use addr=localhost:6379 for development and empty password
func NewRedisClient(ctx context.Context, addr, password string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr, //localhost:6379 for dev/test
		DB:       0,    // use default DB
		Password: password,
		Protocol: 2,
	})

	_, err := rdb.Ping(ctx).Result()
	return rdb, err
}
