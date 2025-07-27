package main

import (
	"context"
	"os"

	"github.com/bkohler93/game-backend/internal/gateway"
	"github.com/bkohler93/game-backend/internal/message"
	"github.com/bkohler93/game-backend/internal/redis"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

func main() {
	loadEnv()
	port := os.Getenv("PORT")
	redisAddr := os.Getenv("REDIS_ADDR")

	redisClient := redis.NewRedisClient(redisAddr, "", 0)
	mb := message.RedisStreamClient{redisClient}

	g := gateway.NewGateway(port, redisClient)

	g.Start(ctx)
}

func loadEnv() {
	if os.Getenv("ENV") != "PROD" {
		godotenv.Load()
	}
}
