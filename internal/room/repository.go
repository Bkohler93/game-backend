package room

import (
	"context"

	"github.com/bkohler93/game-backend/internal/game"
	"github.com/bkohler93/game-backend/internal/message"
)

type Repository struct {
	dao DAO
}

func (r *Repository) QueryOpenRooms(ctx context.Context, req message.MatchmakingRequest) ([]Room, error) {
	minSkill, maxSkill := CalculateMinMaxSkill(req.Skill, req.TimeCreated)
	return r.dao.QueryOpenRooms(ctx, req.Region, minSkill, maxSkill, game.MaxPlayers)
}

func (r *Repository) TryJoinRoom(req message.MatchmakingRequest, room Room) (joinedRoom Room, err error) {
	r.dao.JoinRoom(room.RoomId, req.UserId)
	return Room{}, nil
}

func NewRepository(dao DAO) *Repository {
	return &Repository{
		dao: dao,
	}
}
