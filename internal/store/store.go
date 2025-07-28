package store

import (
	"context"

	"github.com/bkohler93/game-backend/internal/models"
)

type Store interface {
	CreateRoomIndex(context.Context) error
	StoreRoom(context.Context, models.Room) error
	StoreKeyValue(context.Context, string, any) error
}
