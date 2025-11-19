package room

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	PendingTime = time.Second * 1
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
	RoomId              uuidstring.ID
	Skill               int
	TimeCreated         int64
	Region              string
	MaxAllowablePlayers int
}

func InitialQueryRoomRequest(skill int, timeCreated int64, region string) *QueryRoomRequest {
	return &QueryRoomRequest{
		Skill:               skill,
		TimeCreated:         timeCreated,
		Region:              region,
		MaxAllowablePlayers: 2, //TODO if expanding this infrastructure to multiple games this value will need to be set differently somehow.
		//TODO probably not be a magic number anyways. Create const  somewhere?
	}
}

type JoinRoomRequest struct {
	UserId uuidstring.ID
	Skill  int
}
