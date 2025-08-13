package rediskeys

import (
	"fmt"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	MatchmakeTaskPendingSortedSetKey    = "matchmake:task:pending"
	MatchmakeTaskInProgressSortedSetKey = "matchmake:task:inprogress"
)

func RoomsJSONObject(roomId uuidstring.ID) string {
	return fmt.Sprintf("rooms:%s", roomId)
}

func PlayerString(playerId uuidstring.ID) string {
	return fmt.Sprintf("players:%s", playerId)
}

var MatchmakingServerMessageCGroup = fmt.Sprintf("%s:server_message:cgroup", matchmakeStream)

func RoomLockKey(roomId uuidstring.ID) string {
	return fmt.Sprintf("lock:room:%s", roomId)
}

func MatchmakingClientMessageCGroup(clientID uuidstring.ID) string {
	return fmt.Sprintf("%s:client_message:cgroup:%s", matchmakeStream, clientID)
}
