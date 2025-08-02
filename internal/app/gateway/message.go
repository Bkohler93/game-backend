package gateway

import (
	"encoding/json"
)

type ServiceType string

const (
	MatchmakingService ServiceType = "MatchmakingService"
	GameService        ServiceType = "GameService"
)

// BaseMessage is the top-level structure all messages to Client<->Websocket messages must conform to.
// It allows us to read the 'Type' without fully decoding the 'Payload'.
type BaseMessage struct {
	ServiceType ServiceType     `json:"type"`
	Payload     json.RawMessage `json:"payload"` // json.RawMessage holds the raw JSON bytes
}
