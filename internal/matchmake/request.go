package matchmake

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type MatchRequest struct {
	UserId       stringuuid.StringUUID `redis:"user_id" json:"user_id"`
	Name         string                `redis:"name"`
	MatchedWith  stringuuid.StringUUID `redis:"matched_with" json:"matched_with"`
	TimeReceived time.Time             `redis:"time_received" json:"time_received"`
}
