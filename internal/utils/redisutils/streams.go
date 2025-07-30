package redisutils

import "github.com/bkohler93/game-backend/pkg/stringuuid"

const (
	MatchmakeStream = "matchmake"
	GameStream      = "game"
)

func GameServerMessageStream() string {
	return GameStream + ":" + "server_message"
}

func GameClientMessageStream(clientID stringuuid.StringUUID) string {
	return GameStream + ":" + "client_message" + ":" + clientID.String()
}

func MatchmakingServerMessageStream() string {
	return MatchmakeStream + ":" + "server_message"
}

func MatchmakingClientMessageStream(clientID stringuuid.StringUUID) string {
	return MatchmakeStream + ":" + "client_message" + ":" + clientID.String()
}

func MatchmakingRoomEventsStream() string {
	return MatchmakeStream + ":" + "room_events"
}
