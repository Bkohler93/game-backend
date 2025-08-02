package utils

import (
	"context"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	if os.Getenv("ENV") != "PROD" {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	}
}

func LoadLuaSrc(filePath string) (string, error) {
	bytes, err := os.ReadFile(filePath)
	return string(bytes), err
}

func SliceForeachContext[T any](ctx context.Context, items []T, do func(ctx context.Context, item T)) {
	for _, i := range items {
		select {
		case <-ctx.Done():
			return
		default:
			do(ctx, i)
		}
	}
}
