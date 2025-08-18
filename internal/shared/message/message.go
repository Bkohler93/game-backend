package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

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
	//Identifiable
	//AckFuncAccessor
	//MetaDataAccessor
}

type Discriminable interface {
	GetDiscriminator() string
}

type AckFuncAccessor interface {
	Ack(context.Context) error
	SetAck(func(context.Context) error)
}

type Identifiable interface {
	IDGettable
	IDSettable
}

type IDGettable interface {
	GetID() string
}

type IDSettable interface {
	SetID(string)
}

type EmptyMessage struct{}

func (e EmptyMessage) GetDiscriminator() string {
	return ""
}

func (e EmptyMessage) Ack(ctx context.Context) error {
	return nil
}

func (e EmptyMessage) SetAck(f func(context.Context) error) {
	return
}

func (e EmptyMessage) GetMetaData() metadata.MetaData {
	return map[string]string{}
}

func (e EmptyMessage) SetMetaData(m metadata.MetaData) {
}

var PrintTypeDiscriminator = func(i any) string {
	return reflect.TypeOf(i).String()
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
	Type     string            `json:"type"`
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
		Type:    string(msgType),
		Payload: Payload,
	}
}

func (e *Envelope) ToMap() map[string]any {
	return map[string]any{
		"type":    e.Type,
		"payload": string(e.Payload),
	}
}

func (e *Envelope) FromMap(data map[string]any) error {
	dataType, ok := data["type"].(string)
	if !ok {
		return errors.New("invalid stored in 'type' property")
	}
	payload, ok := data["payload"].(json.RawMessage)
	if !ok {
		return errors.New("invalid stored in 'payload' property")
	}
	e.Type = dataType
	e.Payload = payload
	return nil
}

func DecodePayload[T any](payload json.RawMessage) (*T, error) {
	var msg T
	err := json.Unmarshal(payload, &msg)
	return &msg, err
}

type MapSerializable interface {
	ToMap() map[string]interface{}
	FromMap(map[string]interface{}) error
}
