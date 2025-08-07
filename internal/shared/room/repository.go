package room

import (
	"context"
)

type Repository struct {
	store Store
}

func (r *Repository) QueryOpenRooms(ctx context.Context, req QueryRoomRequest) ([]Room, error) {
	minSkill, maxSkill := CalculateMinMaxSkill(req.Skill, req.TimeCreated)
	return r.store.QueryOpenRooms(ctx, req.Region, minSkill, maxSkill, 2) //TODO the '2' magic number should be a constant used for whatever game is being queried for
}

func (r *Repository) JoinRoom(ctx context.Context, req JoinRoomRequest, room Room) (Room, error) {
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
