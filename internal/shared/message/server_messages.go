package message

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type ServerMessageType string

const (
	Matchmaking ServerMessageType = "Matchmaking"
	Game        ServerMessageType = "Game"
)

type MatchmakingServerMessageType string

const (
	RequestMatchmaking MatchmakingServerMessageType = "RequestMatchmaking"
	ExitMatchmaking    MatchmakingServerMessageType = "ExitMatchmaking"
)

//type BaseMatchmakingServerMessage struct {
//	Type    MatchmakingServerMessageType `json:"type"`
//	Payload json.RawMessage              `json:"payload"` // json.RawMessage holds the raw JSON bytes
//}
//
//func (b BaseMatchmakingServerMessage) ToMap() map[string]interface{} {
//}
//
//func (b BaseMatchmakingServerMessage) FromMap(m map[string]interface{}) error {
//	//TODO implement me
//	panic("implement me")
//}

type MatchmakingRequest struct {
	UserId      stringuuid.StringUUID `redis:"user_id" json:"user_id"`
	Name        string                `redis:"name" json:"name"`
	TimeCreated time.Time             `redis:"time_created" json:"time_created"` //TODO remove this? The Room object will contain a Retry count
	Skill       int                   `redis:"skill" json:"skill"`
	Region      string                `redis:"region" json:"region"`
}

type MatchmakingExit struct {
	UserId stringuuid.StringUUID `json:"user_id"`
}

type GameServerMessageType string

//type BaseGameServerMessage struct {
//	Type    GameServerMessageType `json:"type"`
//	Payload json.RawMessage       `json:"payload"` // json.RawMessage holds the raw JSON bytes
//}
