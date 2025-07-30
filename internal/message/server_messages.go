package message

import (
	"encoding/json"
	"time"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type MatchmakingServerMessageType string

const (
	ServerMessageTypeMatchmakingRequest MatchmakingServerMessageType = "MatchmakingRequest"
	ServerMessageTypeMatchmakingExit    MatchmakingServerMessageType = "MatchmakingExit"
)

type BaseMatchmakingServerMessage struct {
	Type    MatchmakingServerMessageType `json:"type"`
	Payload json.RawMessage              `json:"payload"` // json.RawMessage holds the raw JSON bytes
}

type MatchmakingRequest struct {
	UserId      stringuuid.StringUUID `redis:"user_id" json:"user_id"`
	Name        string                `redis:"name" json:"name"`
	TimeCreated time.Time             `redis:"time_created" json:"time_created"`
	Skill       int                   `redis:"skill" json:"skill"`
	Region      string                `redis:"region" json:"region"`
}

type ExitMatchmaking struct {
	UserId stringuuid.StringUUID `json:"user_id"`
}
