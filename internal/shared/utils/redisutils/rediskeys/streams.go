package rediskeys

import (
	"fmt"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	matchmakeStream              = "matchmake"
	gameStream                   = "game"
	MatchmakeNotifyWorkersPubSub = "matchmake:notify_worker"
)

func GameServerMessageStream(roomId uuidstring.ID) string {
	return fmt.Sprintf("%s:server_message:%s", gameStream, roomId)
}

func GameClientMessageStream(clientId uuidstring.ID) string {
	return fmt.Sprintf("%s:client_message:%s", gameStream, clientId)
}

var MatchmakingServerMessageStream = fmt.Sprintf("%s:server_message", matchmakeStream)

func MatchmakingClientMessageStream(clientId uuidstring.ID) string {
	return fmt.Sprintf("%s:client_message:%s", matchmakeStream, clientId)
}

var MatchmakingRoomEventsStream = fmt.Sprintf("%s:room:events", matchmakeStream)
