package files

// File: internal/shared/utils/redisutils/scripts.go

import (
	"embed"
	"fmt"
)

const (
	LuaAddPlayerToRoom           = "add_player_to_room.lua"
	LuaCGroupAckDelMsg           = "cgroup_xack_xdel_atomic.lua"
	LuaClaimNextPendingTask      = "claim_next_pending_task.lua"
	LuaStaleMatchmakingToPending = "stale_matchmaking_to_pending.lua"
	LuaUnlockRoom                = "unlock_room.lua"
	LuaCombineRooms              = "combine_rooms.lua"
	LuaRemovePlayerFromRoom      = "remove_player_from_room.lua"
	LuaZPopOldestFromSortedSet   = "zpop_oldest_from_sorted_set.lua"
)

//go:embed db/redis/scripts/*.lua
var LuaScripts embed.FS

func GetLuaScript(name string) (string, error) {
	content, err := LuaScripts.ReadFile("db/redis/scripts/" + name)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded Lua script: %s, error: %v", name, err)
	}
	return string(content), nil
}
