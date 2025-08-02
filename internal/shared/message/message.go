package message

import (
	"context"
	"reflect"
)

var PrintTypeDiscriminator = func(i any) string {
	return reflect.TypeOf(i).String()
}

type Producer interface {
	Publish(ctx context.Context, channel string, data any) error
}

type Consumer interface {
	Consume(ctx context.Context, channel string, output any) error
}
