package transport

import (
	"context"
	"fmt"

	"github.com/bkohler93/game-backend/internal/shared/message"
)

func UnwrapAndForward[T message.Message](ctx context.Context, wrappedMsgCh <-chan WrappedConsumeMsg, errCh <-chan error, serverMessageTypeConstructorRegistry map[string]func() T) (<-chan T, <-chan error) {
	msgCh := make(chan T)
	go func() {
		for {
			select {
			case wrappedMsg, open := <-wrappedMsgCh:
				if !open {
					return
				}
				bytes := wrappedMsg.Payload.([]byte)

				msg, err := message.UnmarshalWrappedType[T](bytes, serverMessageTypeConstructorRegistry)
				if err != nil {
					fmt.Printf("received invalid msg payload - %v", err)
					continue
				}

				msgCh <- msg
			case err := <-errCh:
				fmt.Printf("received error from ServerMessageConsumer - %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgCh, errCh
}
