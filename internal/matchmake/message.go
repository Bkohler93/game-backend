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

const MatchmakingTypeDiscriminator = "MatchmakingResponse"

type MatchResponse struct {
	TypeDiscriminator string                `json:"$type"`
	UserOneId         stringuuid.StringUUID `redis:"user_one_id" json:"user_one_id"`
	UserOneName       string                `redis:"user_one_name" json:"user_one_name"`
	UserTwoId         stringuuid.StringUUID `redis:"user_two_id" json:"user_two_id"`
	UserTwoName       string                `redis:"user_two_name" json:"user_two_name"`
}

func NewMatchmakingResponse() MatchResponse {
	return MatchResponse{
		TypeDiscriminator: MatchmakingTypeDiscriminator,
	}
}
