package main

import (
	"context"
	"os"

	"github.com/bkohler93/game-backend/internal/gateway"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

func main() {
	loadEnv()
	port := os.Getenv("PORT")
	redisAddr := os.Getenv("REDIS_ADDR")
	g := gateway.NewGateway(port, redisAddr)

	g.Start(ctx)
}

func loadEnv() {
	if os.Getenv("ENV") != "PROD" {
		godotenv.Load()
	}
}
