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

func (r *Repository) DeleteRoom(ctx context.Context, roomId uuidstring.ID) error {
	return r.store.DeleteRoom(ctx, roomId)
}

func (r *Repository) QueryOpenRooms(ctx context.Context, req *QueryRoomRequest) ([]Room, error) {
	minSkill, maxSkill := CalculateMinMaxSkill(req.Skill, req.TimeCreated)

	rooms, err := r.store.QueryOpenRooms(ctx, req.Region, minSkill, maxSkill, req.MaxAllowablePlayers)
	filtered := rooms[:0]
	for _, rm := range rooms {
		if rm.RoomId != req.RoomId {
			filtered = append(filtered, rm)
		}
	}
	return filtered, err
}

func (r *Repository) JoinRoom(ctx context.Context, req JoinRoomRequest, room Room) (Room, error) {
	return r.store.JoinRoom(ctx, room.RoomId, req.UserId, req.Skill)
}

func (r *Repository) CreateRoom(ctx context.Context, room Room) error {
	return r.store.InsertRoom(ctx, room)
}

func (r *Repository) CreateRoomIndex(ctx context.Context) error {
	err := r.store.CreateRoomIndex(ctx)
	if err != nil {
		if err.Error() == "index already exists" || err.Error() == "Index already exists" {
			return nil
		}
		return err
	}
	return nil
}

func (r *Repository) GetRoom(ctx context.Context, roomID uuidstring.ID) (Room, error) {
	return r.store.GetRoom(ctx, roomID)
}

// LockRoom returns ErrDidNotLock if unable to lock
func (r *Repository) LockRoom(ctx context.Context, id uuidstring.ID) (uuidstring.ID, error) {
	return r.store.LockRoom(ctx, id)
}

func (r *Repository) UnlockRoom(ctx context.Context, id uuidstring.ID, key uuidstring.ID) error {
	return r.store.UnlockRoom(ctx, id, key)
}

// CombineRooms combines room1 into room2, returning the resulting room with same id as room2
func (r *Repository) CombineRooms(ctx context.Context, roomOneId uuidstring.ID, roomTwoId uuidstring.ID) (combinedRoom Room, err error) {
	combinedRoom, err = r.store.CombineRooms(ctx, roomOneId, roomTwoId)
	return
}

func (r *Repository) RemovePlayer(ctx context.Context, roomId uuidstring.ID, userId uuidstring.ID, userSkill int) ([]uuidstring.ID, error) {
	rm, err := r.store.RemovePlayer(ctx, roomId, userId, userSkill)
	if err != nil {
		return rm, err
	}
	//TODO remove player and delete room if were the only one in it
	return rm, err
}

func NewRepository(ctx context.Context, store Store) (*Repository, error) {
	r := &Repository{
		store: store,
	}
	err := r.CreateRoomIndex(ctx)
	return r, err
}
