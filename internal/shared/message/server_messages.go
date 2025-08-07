package message

type ServerMessageType string

const (
	Matchmaking ServerMessageType = "Matchmaking"
	Game        ServerMessageType = "Game"
)

type GameServerMessageType string

//type BaseGameServerMessage struct {
//	Type    GameServerMessageType `json:"type"`
//	Payload json.RawMessage       `json:"payload"` // json.RawMessage holds the raw JSON bytes
//}
