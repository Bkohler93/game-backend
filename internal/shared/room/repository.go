package room

import (
	"context"
	"errors"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

var (
	ErrRoomFull = errors.New("room is full")
)

type Repository struct {
	store Store
}

func (r *Repository) QueryOpenRooms(ctx context.Context, req *QueryRoomRequest) ([]Room, error) {
	minSkill, maxSkill := CalculateMinMaxSkill(req.Skill, req.TimeCreated)
	return r.store.QueryOpenRooms(ctx, req.RoomId, req.Region, minSkill, maxSkill, 2) //TODO the '2' magic number should be a constant used for whatever game is being queried for
}

func (r *Repository) JoinRoom(ctx context.Context, req JoinRoomRequest, room Room) (Room, error) {
	return r.store.JoinRoom(ctx, room.RoomId, req.UserId, req.Skill)
}

func (r *Repository) CreateRoom(ctx context.Context, room Room) error {
	return r.store.InsertRoom(ctx, room)
}

func (r *Repository) CreateRoomIndex(ctx context.Context) error {
	return r.store.CreateRoomIndex(ctx)
}

func (r *Repository) GetRoom(ctx context.Context, roomID uuidstring.ID) (Room, error) {
	return r.store.GetRoom(ctx, roomID)
}

func (r *Repository) LockRoom(ctx context.Context, id uuidstring.ID) (uuidstring.ID, error) {
	return r.store.LockRoom(ctx, id)
}

func (r *Repository) UnlockRoom(ctx context.Context, id uuidstring.ID, key uuidstring.ID) error {
	return r.store.UnlockRoom(ctx, id, key)
}

// CombineRooms combines room1 into room2, returning the resulting room with same id as room2
func (r *Repository) CombineRooms(ctx context.Context, room1 Room, room2 Room) (combinedRoom Room, err error) {
	combinedRoom, err = r.store.CombineRooms(ctx, room1, room2)
	return
}

func NewRepository(store Store) *Repository {
	return &Repository{
		store: store,
	}
}
