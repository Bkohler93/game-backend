package main

import (
	"context"
	"os"

	"github.com/bkohler93/game-backend/internal/gateway"
	"github.com/bkohler93/game-backend/internal/message"
	"github.com/bkohler93/game-backend/internal/store"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	loadEnv()
	port := os.Getenv("PORT")
	redisAddr := os.Getenv("REDIS_ADDR")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		DB:       0, // use default DB
		Password: "",
	})

	mb := message.NewRedisStreamClient(redisClient)
	s := store.NewRediStore(redisClient)

	g := gateway.NewGateway(port, mb, s)

	g.Start(ctx)
}

func loadEnv() {
	if os.Getenv("ENV") != "PROD" {
		godotenv.Load()
	}
}
