package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bkohler93/game-backend/internal/app/auth"
	"github.com/bkohler93/game-backend/internal/shared/utils"
)

func main() {
	utils.LoadEnv()
	port := os.Getenv("AUTH_PORT")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	a := auth.NewAuthServer(port)
	a.Start(ctx)
}
