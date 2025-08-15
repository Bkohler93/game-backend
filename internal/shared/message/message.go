package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type ServiceType string

const (
	MatchmakingService ServiceType = "MatchmakingService"
	GameService        ServiceType = "GameService"
)

type Message interface {
	Discriminable
	//Identifiable
	AckFuncAccessor
	MetaDataAccessor
}

type MetaDataAccessor interface {
	GetMetaData() map[string]string
	SetMetaData(map[string]string)
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

var PrintTypeDiscriminator = func(i any) string {
	return reflect.TypeOf(i).String()
}

func UnmarshalWrappedType[T Discriminable](data []byte, typeRegistry map[string]func() T) (T, error) {
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
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"` // json.RawMessage holds the raw JSON bytes
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
