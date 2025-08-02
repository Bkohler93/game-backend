package redisutils

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient
// use addr=localhost:6379 for development and empty password, uses env variables to fill these values, be sure to use
// utils.LoadEnv() before creating the client if you are in a dev/test environment
func NewRedisClient(ctx context.Context) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"), //localhost:6379 for dev/test
		DB:       0,                       // use default DB
		Password: os.Getenv("REDIS_PW"),
		Protocol: 2,
	})

	_, err := rdb.Ping(ctx).Result()
	return rdb, err
}
