package taskcoordinator

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type MatchmakingTaskStore interface {
	AddPendingTask(ctx context.Context, roomID uuidstring.ID, StartProcessingTime int64) error
	RemovePendingTask(ctx context.Context, roomID uuidstring.ID) error
	ClaimPendingTask(ctx context.Context) (uuidstring.ID, error)
	AddInProgressTask(ctx context.Context, roomID uuidstring.ID, ProcessingStartedTime int64) error
	RemoveInProgressTask(ctx context.Context, roomID uuidstring.ID) error
	GetStaleInProgressTasks(ctx context.Context, OldestAllowableTime int64) ([]uuidstring.ID, error)
	MoveInProgressToPendingTask(ctx context.Context, roomID uuidstring.ID) error
}
