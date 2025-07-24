package gateway

import (
	"encoding/json"
)

type MessageType string

const (
	MessageTypeMatchmaking MessageType = "Matchmaking"
	MessageTypeGameplay    MessageType = "Gameplay"
)

// BaseMessage is the top-level structure all incoming WebSocket messages must conform to.
// It allows us to read the 'Type' without fully decoding the 'Payload'.
type BaseMessage struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"` // json.RawMessage holds the raw JSON bytes
}
