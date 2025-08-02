package room

import (
	"context"

	"github.com/bkohler93/game-backend/internal/app/game"
	"github.com/bkohler93/game-backend/internal/shared/message"
)

type Repository struct {
	store Store
}

func (r *Repository) QueryOpenRooms(ctx context.Context, req message.MatchmakingRequest) ([]Room, error) {
	minSkill, maxSkill := CalculateMinMaxSkill(req.Skill, req.TimeCreated)
	return r.store.QueryOpenRooms(ctx, req.Region, minSkill, maxSkill, game.MaxPlayers)
}

func (r *Repository) JoinRoom(ctx context.Context, req message.MatchmakingRequest, room Room) (Room, error) {
	return r.store.JoinRoom(ctx, room.RoomId, req.UserId, req.Skill)
}

func (r *Repository) CreateRoom(ctx context.Context, room Room) error {
	return r.store.InsertRoom(ctx, room)
}

func NewRepository(store Store) *Repository {
	return &Repository{
		store: store,
	}
}
