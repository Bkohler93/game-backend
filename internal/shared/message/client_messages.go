package message

import (
	"encoding/json"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type MatchmakingClientMessageType string

const (
	ClientMessageTypeMatchMade        MatchmakingClientMessageType = "MatchMade"
	ClientMessageTypePlayerLeftRoom   MatchmakingClientMessageType = "PlayerLeftRoom"
	ClientMessageTypePlayerJoinedRoom MatchmakingClientMessageType = "PlayerJoinedRoom"
	ClientMessageTypeRoomChanged      MatchmakingClientMessageType = "RoomChanged"
)

type BaseMatchmakingClientMessage struct {
	Type    MatchmakingClientMessageType `json:"type"`
	Payload json.RawMessage              `json:"payload"`
}

type BaseGameClientMessage struct {
	Type    GameClientMessageType `json:"type"`
	Payload json.RawMessage       `json:"payload"`
}

type GameClientActionBase struct {
	GameID   stringuuid.StringUUID
	ClientID stringuuid.StringUUID
}
