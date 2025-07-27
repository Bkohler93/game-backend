package message

import "context"

type MessageBus interface {
	Producer
	Consumer
}

type Producer interface {
	Publish(ctx context.Context, channel string, data any) error
}

type Consumer interface {
	Consume(ctx context.Context, channel string, output any) error
}
