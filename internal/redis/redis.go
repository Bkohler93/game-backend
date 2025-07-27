package redis

import (
	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

const (
	MatchmakeStream        = "matchmake"
	MatchfoundStream       = MatchmakeStream + ":found"
	MatchmakePool          = "matchmake:pool"
	MatchmakeRequestStream = MatchmakeStream + ":request"
	GameStream             = "game"
)

func MatchFoundStream(userId stringuuid.StringUUID) string {
	return MatchfoundStream + ":" + userId.String()
}

func MatchmakePoolUser(userId stringuuid.StringUUID) string {
	return MatchmakePool + ":" + userId.String()
}

var AllMatchmakePool = MatchmakePool + ":*"

func GameClientActionStream(gameId stringuuid.StringUUID) string {
	return GameStream + ":" + gameId.String() + ":client:action"
}

func GameServerResponseStream(userId stringuuid.StringUUID) string {
	return GameStream + ":server:response:" + userId.String()
}
