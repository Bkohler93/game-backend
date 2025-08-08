package files

import (
	"testing"
)

func TestGetScript(t *testing.T) {
	src, err := GetLuaScript("add_player_to_room.lua")
	if err != nil {
		t.Errorf("unexpected error - %v", err)
	}
	if src == "" {
		t.Errorf("expected src to have contents - %v", src)
	}
}
