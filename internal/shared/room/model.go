package room

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	PendingTime = time.Second * 5
)

type Room struct {
	RoomId            uuidstring.ID   `json:"room_id"`
	PlayerCount       int             `json:"player_count"`
	AverageSkill      int             `json:"average_skill"`
	Region            string          `json:"region"`
	PlayerIds         []uuidstring.ID `json:"player_ids"`
	CreatedAt         int64           `json:"created_at"`
	MatchmakeAttempts int             `json:"matchmake_attempts"`
	IsFull            int             `json:"is_full"`
}

type QueryRoomRequest struct {
	Skill       int
	TimeCreated int64
	Region      string
}

type JoinRoomRequest struct {
	UserId uuidstring.ID
	Skill  int
}
