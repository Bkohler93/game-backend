package message

import (
	"encoding/json"
	"errors"
	"reflect"
)

type Discriminable interface {
	GetDiscriminator() string
}

var PrintTypeDiscriminator = func(i any) string {
	return reflect.TypeOf(i).String()
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
