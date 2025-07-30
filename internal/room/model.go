package room

import (
	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type Room struct {
	RoomId       stringuuid.StringUUID   `json:"room_id"`
	PlayerCount  int                     `json:"player_count"`
	AverageSkill int                     `json:"average_skill"`
	Region       string                  `json:"region"`
	PlayerIds    []stringuuid.StringUUID `json:"player_ids"`
	CreatedAt    int64                   `json:"created_at"`
	IsFull       int                     `json:"is_full"`
}
