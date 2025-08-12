package taskcoordinator

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type MatchmakingTaskCoordinator struct {
	store MatchmakingTaskStore
}

func NewMatchmakingTaskCoordinator(store MatchmakingTaskStore) *MatchmakingTaskCoordinator {
	return &MatchmakingTaskCoordinator{
		store: store,
	}
}

func (c *MatchmakingTaskCoordinator) ClaimNextPendingTask(ctx context.Context) (uuidstring.ID, error) {
	roomID, err := c.store.ClaimPendingTask(ctx)
	if err != nil {
		return "", err
	}
	return roomID, err
}

func (c *MatchmakingTaskCoordinator) AddPendingTask(ctx context.Context, roomId uuidstring.ID, startProcessingTime int64) error {
	return c.store.AddPendingTask(ctx, roomId, startProcessingTime)
}

func (c *MatchmakingTaskCoordinator) ReclaimStaleInProgressTasks(ctx context.Context, cutoff int64) ([]uuidstring.ID, error) {
	stale, err := c.store.GetStaleInProgressTasks(ctx, cutoff)
	if err != nil {
		return nil, err
	}

	for _, roomID := range stale {
		err = c.store.MoveInProgressToPendingTask(ctx, roomID)
		if err != nil {

		}
	}
	return stale, nil
}
