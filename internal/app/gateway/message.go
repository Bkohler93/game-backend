package gateway

import (
	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type GatewayServerMessage interface {
	message.Message
}

type GatewayServerMessageType string

const (
	ConnectingMessageType GatewayServerMessageType = "ConnectingMessage"
)

type ConnectingMessage struct {
	JwtString string `json:"jwt_string"`
}

func (c ConnectingMessage) GetDiscriminator() string {
	return "ConnectingMessage"
}

// var gatewayServerMessageRegistry = map[string]func() GatewayServerMessage{
// 	string(ConnectingMessageType): func() GatewayServerMessage { return &ConnectingMessage{} },
// }

type GatewayClientMessage interface {
	message.Message
}

type GatewayClientMessageType string

const (
	Authenticated GatewayClientMessageType = "AuthenticatedMessage"
)

type AuthenticatedMessage struct {
	TypeDiscriminator string        `json:"$type"`
	UserId            uuidstring.ID `json:"user_id"`
}

func NewAuthenticatedMessage(userId uuidstring.ID) *AuthenticatedMessage {
	return &AuthenticatedMessage{
		TypeDiscriminator: string(Authenticated),
		UserId:            userId,
	}
}

func (a AuthenticatedMessage) GetDiscriminator() string {
	return a.TypeDiscriminator
}

// var gatewayClientMessageRegistry = map[string]func() GatewayClientMessage{
// 	string(Authenticated): func() GatewayClientMessage { return &AuthenticatedMessage{} },
// }
