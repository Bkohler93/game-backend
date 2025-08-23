package message

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bkohler93/game-backend/internal/shared/constants/metadata"
)

type ServiceType string

const (
	MatchmakingService ServiceType = "MatchmakingService"
	SetupService       ServiceType = "SetupService"
	GameService        ServiceType = "GameService"
)

type EnvelopeContext struct {
	Env     *Envelope
	AckFunc func(ctx context.Context) error
}

func NewNoAckEnvelopeContext(e *Envelope) *EnvelopeContext {
	return &EnvelopeContext{
		Env: e,
		AckFunc: func(ctx context.Context) error {
			return nil
		},
	}
}

type MessageContext struct {
	Msg     Message
	AckFunc func(ctx context.Context) error
}

type Message interface {
	Discriminable
}

type Discriminable interface {
	GetDiscriminator() string
}

type AckFuncAccessor interface {
	Ack(context.Context) error
	SetAck(func(context.Context) error)
}

type EmptyMessage struct{}

func (e EmptyMessage) GetDiscriminator() string {
	return ""
}

func UnmarshalWrappedType[T Message](data []byte, typeRegistry map[string]func() T) (T, error) {
	var zeroValue T
	var temp struct {
		TypeDiscriminator string `json:"$type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return zeroValue, err
	}

	if constructor, ok := typeRegistry[temp.TypeDiscriminator]; ok {
		concreteMessage := constructor()
		if err := json.Unmarshal(data, &concreteMessage); err != nil {
			return concreteMessage, err
		}
		return concreteMessage, nil
	}

	return zeroValue, fmt.Errorf("unknown matchmaking message type: %s", temp.TypeDiscriminator)
}

type Envelope struct {
	Type     ServiceType       `json:"type"`
	Payload  json.RawMessage   `json:"payload"` // json.RawMessage holds the raw JSON bytes
	MetaData metadata.MetaData `json:"metadata"`
}

func (e *Envelope) EnsureMetaData() {
	if e.MetaData == nil {
		e.MetaData = make(metadata.MetaData)
	}
}

func NewEnvelopeOf[T ~string](msgType T, Payload json.RawMessage) Envelope {
	return Envelope{
		Type:    ServiceType(msgType),
		Payload: Payload,
	}
}
