package store

import "github.com/redis/go-redis/v9"

type RedisStore struct {
	*redis.Client
}

func NewRediStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{
		rdb,
	}
}
