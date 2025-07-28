package models

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type Room struct {
	RoomId       stringuuid.StringUUID   `json:"room_id"`
	PlayerCount  int                     `json:"player_count"`
	AverageSkill int                     `json:"average_skill"`
	Region       string                  `json:"region"`
	PlayerIds    []stringuuid.StringUUID `json:"player_ids"`
	CreatedAt    int64                   `json:"created_at"`
}

func RandomRoom() Room {
	id := stringuuid.NewStringUUID()
	return Room{
		RoomId:       id,
		PlayerCount:  1,
		AverageSkill: 100,
		Region:       "na",
		PlayerIds:    []stringuuid.StringUUID{id},
		CreatedAt:    time.Now().Unix(),
	}
}
