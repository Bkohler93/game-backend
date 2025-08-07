package message

import (
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

//type BaseMatchmakingClientMessage struct {
//	Type    MatchmakingClientMessageType `json:"type"`
//	Payload json.RawMessage              `json:"payload"`
//}

//func (m *BaseMatchmakingClientMessage) ToMap() map[string]interface{} {
//	return map[string]interface{}{
//		"type":    string(m.Type),
//		"payload": []byte(m.Payload),
//	}
//}

//func (m *BaseMatchmakingClientMessage) FromMap(data map[string]interface{}) error {
//	msgType, ok := data["type"].(string)
//	if !ok {
//		return fmt.Errorf("field 'type' is not a string")
//	}
//	m.Type = MatchmakingClientMessageType(msgType)
//
//	payloadString, ok := data["payload"].(string)
//	if !ok {
//		return fmt.Errorf("field 'payload' is not a string")
//	}
//
//	// The key here is that `payloadString` is a Go string that contains
//	// the raw JSON. We use `json.RawMessage` to convert it back to a []byte.
//	m.Payload = json.RawMessage(payloadString)
//
//	return nil
//}

type GameClientMessageType string

//type BaseGameClientMessage struct {
//	Type    GameClientMessageType `json:"type"`
//	Payload json.RawMessage       `json:"payload"`
//}

type GameClientActionBase struct {
	GameID   uuidstring.ID
	ClientID uuidstring.ID
}
