package main

import (
	"context"
	"os"

	"github.com/bkohler93/game-backend/internal/matchmake"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

func main() {
	loadEnv()
	redisAddr := os.Getenv("REDIS_ADDR")

	m := matchmake.NewMatchmaker(redisAddr)
	m.Start(ctx)
}

func loadEnv() {
	if os.Getenv("ENV") != "PROD" {
		godotenv.Load()
	}
}
