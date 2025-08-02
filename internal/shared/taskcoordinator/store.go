package taskcoordinator

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type MatchmakingTaskStore interface {
	AddPendingTask(ctx context.Context, roomID stringuuid.StringUUID, StartProcessingTime int64) error
	RemovePendingTask(ctx context.Context, roomID stringuuid.StringUUID) error
	ClaimPendingTask(ctx context.Context) (stringuuid.StringUUID, error)
	AddInProgressTask(ctx context.Context, roomID stringuuid.StringUUID, ProcessingStartedTime int64) error
	RemoveInProgressTask(ctx context.Context, roomID stringuuid.StringUUID) error
	GetStaleInProgressTasks(ctx context.Context, OldestAllowableTime int64) ([]stringuuid.StringUUID, error)
	MoveInProgressToPendingTask(ctx context.Context, roomID stringuuid.StringUUID) error
}
