package rediskeys

import (
	"fmt"
)

const (
	matchmakeStream              = "matchmake"
	gameStream                   = "game"
	MatchmakeNotifyWorkersPubSub = "matchmake:notify_worker"
)

var GameServerMessageStream = fmt.Sprintf("%s:server_message", gameStream)
var GameClientMessageStream = fmt.Sprintf("%s:client_message", gameStream)

func GameServerMessageStreamDestination(gameServerInstanceHostname string) string {
	return fmt.Sprintf("%s:%s", GameServerMessageStream, gameServerInstanceHostname)
}

func GameClientMessageStreamDestination(wsGatewayInstanceHostname string) string {
	return fmt.Sprintf("%s:%s", GameClientMessageStream, wsGatewayInstanceHostname)
}

var MatchmakingServerMessageStream = fmt.Sprintf("%s:server_message", matchmakeStream)

func MatchmakingServerMessageStreamDestination(matchmakingInstanceHostname string) string {
	return fmt.Sprintf("%s:%s", MatchmakingServerMessageStream, matchmakingInstanceHostname)
}

var MatchmakingClientMessageStream = fmt.Sprintf("%s:client_message", matchmakeStream)

func MatchmakingClientMessageStreamDestination(wsGatewayInstanceHostname string) string {
	return fmt.Sprintf("%s:%s", MatchmakingClientMessageStream, wsGatewayInstanceHostname)
}

var MatchmakingRoomEventsStream = fmt.Sprintf("%s:room:events", matchmakeStream)
