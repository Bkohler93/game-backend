package redis

import (
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

const (
	MatchmakeStream        = "matchmake"
	MatchfoundStream       = MatchmakeStream + ":found"
	MatchmakePool          = "matchmake:pool"
	MatchmakeRequestStream = MatchmakeStream + ":request"
	GameStream             = "game"
)

func MatchFoundStream(userId stringuuid.StringUUID) string {
	return MatchfoundStream + ":" + userId.String()
}

func MatchmakePoolUser(userId stringuuid.StringUUID) string {
	return MatchmakePool + ":" + userId.String()
}

var AllMatchmakePool = MatchmakePool + ":*"

func GameClientActionStream(gameId stringuuid.StringUUID) string {
	return GameStream + ":" + gameId.String() + ":client:action"
}

func GameServerResponseStream(userId stringuuid.StringUUID) string {
	return GameStream + ":server:response:" + userId.String()
}

type RedisClient struct {
	*redis.Client
}

func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	return &RedisClient{
		rdb,
	}
}
