package taskcoordinator

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type GameTaskStore interface {
	AddNewGameTask(ctx context.Context, roomID uuidstring.ID) error
	RemoveOldestNewGameTask(ctx context.Context) (uuidstring.ID, error)
}
