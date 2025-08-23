package taskcoordinator

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type GameTaskCoordinator struct {
	store GameTaskStore
}

func NewGameTaskCoordinator(store GameTaskStore) *GameTaskCoordinator {
	return &GameTaskCoordinator{
		store: store,
	}
}

func (g *GameTaskCoordinator) EnqueueNewGameTask(ctx context.Context, roomID uuidstring.ID) error {
	return g.store.AddNewGameTask(ctx, roomID)
}

func (g *GameTaskCoordinator) DequeueNewGameTask(ctx context.Context) (uuidstring.ID, error) {
	return g.store.RemoveOldestNewGameTask(ctx)
}
