package rediskeys

import (
	"fmt"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

const (
	MatchmakeTaskPendingSortedSetKey    = "matchmake:task:pending"
	MatchmakeTaskInProgressSortedSetKey = "matchmake:task:inprogress"
)

func RoomsJSONObject(roomId stringuuid.StringUUID) string {
	return fmt.Sprintf("rooms:%s", roomId)
}

func PlayerString(playerId stringuuid.StringUUID) string {
	return fmt.Sprintf("players:%s", playerId)
}

var MatchmakingServerMessageCGroup = fmt.Sprintf("%s:server_message:cgroup", matchmakeStream)

func MatchmakingClientMessageCGroup(clientID stringuuid.StringUUID) string {
	return fmt.Sprintf("%s:client_message:cgroup:%s", matchmakeStream, clientID)
}
