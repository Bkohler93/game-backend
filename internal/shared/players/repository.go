package players

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type Repository struct {
	trackerStore TrackerStore
	//TODO userStore will go here, probably
}

func NewRepository(store TrackerStore) *Repository {
	return &Repository{
		trackerStore: store,
	}
}

func (r *Repository) SetPlayerActive(ctx context.Context, playerId stringuuid.StringUUID, roomId stringuuid.StringUUID) error {
	return r.trackerStore.Set(ctx, playerId, roomId)
}

func (r *Repository) GetPlayerRoom(ctx context.Context, playerId stringuuid.StringUUID) (stringuuid.StringUUID, error) {
	return r.trackerStore.Get(ctx, playerId)
}
