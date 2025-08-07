package players

import (
	"context"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
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

func (r *Repository) SetPlayerActive(ctx context.Context, playerId uuidstring.ID, roomId uuidstring.ID) error {
	return r.trackerStore.Set(ctx, playerId, roomId)
}

func (r *Repository) GetPlayerRoom(ctx context.Context, playerId uuidstring.ID) (uuidstring.ID, error) {
	return r.trackerStore.Get(ctx, playerId)
}
