package matchmake

import (
	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type MatchResponse struct {
	UserOneId   stringuuid.StringUUID `redis:"user_one_id" json:"user_one_id"`
	UserOneName string                `redis:"user_one_name" json:"user_one_name"`
	UserTwoId   stringuuid.StringUUID `redis:"user_two_id" json:"user_two_id"`
	UserTwoName string                `redis:"user_two_name" json:"user_two_name"`
}
