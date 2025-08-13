package utils

import (
	"context"
	"errors"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	if os.Getenv("ENV") != "PROD" && os.Getenv("ENV") != "DEV" {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	}
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

func ErrorsIsAny(err error, errs ...error) bool {
	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
