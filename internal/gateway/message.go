package gateway

import "encoding/json"

type ServiceMessageType string

const (
	ServiceMessageTypeMatchmaking ServiceMessageType = "Matchmaking"
	ServiceMessageTypeGameplay    ServiceMessageType = "Game"
)

// BaseMessage is the top-level structure all messages to Client<->Websocket messages must conform to.
// It allows us to read the 'Type' without fully decoding the 'Payload'.
type BaseMessage struct {
	Type    ServiceMessageType `json:"type"`
	Payload json.RawMessage    `json:"payload"` // json.RawMessage holds the raw JSON bytes
}
