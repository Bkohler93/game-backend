package matchmake

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type MatchRequest struct {
	UserId       stringuuid.UserId `redis:"user_id" json:"user_id"`
	Name         string            `redis:"name"`
	MatchedWith  stringuuid.UserId `redis:"matched_with" json:"matched_with"`
	TimeReceived time.Time         `redis:"time_received" json:"time_received"`
}
