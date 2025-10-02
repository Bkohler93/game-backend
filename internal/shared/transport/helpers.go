package transport

import (
	"context"
	"log"

	"github.com/bkohler93/game-backend/internal/shared/message"
)

func UnwrapAndForward[T message.Message](ctx context.Context, envelopeCh <-chan *message.EnvelopeContext, errCh <-chan error, messageTypeConstructorRegistry map[string]func() T) (<-chan *message.MessageContext, <-chan error) {
	msgCh := make(chan *message.MessageContext)
	go func() {
		for {
			select {
			case envelope, open := <-envelopeCh:
				if !open {
					return
				}
				bytes := envelope.Env.Payload

				msg, err := message.UnmarshalWrappedType(bytes, messageTypeConstructorRegistry)
				if err != nil {
					log.Printf("received invalid msg payload - %v", err)
					continue
				}

				msgContext := &message.MessageContext{
					Msg:      msg,
					AckFunc:  envelope.AckFunc,
					Metadata: &envelope.Env.MetaData,
				}

				msgCh <- msgContext
			case err := <-errCh:
				log.Println(err)
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgCh, errCh
}
