package rediskeys

import (
	"fmt"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

const (
	matchmakeStream              = "matchmake"
	gameStream                   = "game"
	MatchmakeNotifyWorkersPubSub = "matchmake:notify_worker"
)

func GameServerMessageStream(gameId stringuuid.StringUUID) string {
	return fmt.Sprintf("%s:server_message:%s", gameStream, gameId)
}

func GameClientMessageStream(clientId stringuuid.StringUUID) string {
	return fmt.Sprintf("%s:client_message:%s", gameStream, clientId)
}

var MatchmakingServerMessageStream = fmt.Sprintf("%s:server_message", matchmakeStream)

func MatchmakingClientMessageStream(clientId stringuuid.StringUUID) string {
	return fmt.Sprintf("%s:client_message:%s", matchmakeStream, clientId)
}

var MatchmakingRoomEventsStream = fmt.Sprintf("%s:room:events", matchmakeStream)
