package transport

import (
	"context"
	"fmt"
	"log"

	"github.com/bkohler93/game-backend/internal/shared/message"
)

func UnwrapAndForward[T message.Message](ctx context.Context, wrappedMsgCh <-chan AckableMessage, errCh <-chan error, messageTypeConstructorRegistry map[string]func() T) (<-chan T, <-chan error) {
	msgCh := make(chan T)
	go func() {
		for {
			select {
			case wrappedMsg, open := <-wrappedMsgCh:
				if !open {
					return
				}
				bytes := wrappedMsg.Payload.([]byte)

				msg, err := message.UnmarshalWrappedType[T](bytes, messageTypeConstructorRegistry)
				if err != nil {
					log.Printf("received invalid msg payload - %v", err)
					continue
				}

				//TODO trying out using AckFunc instead of constructing it using the ID later, keeps ID from being passed to client
				msg.SetAck(wrappedMsg.AckFunc)
				msg.SetMetaData(wrappedMsg.Metadata)
				//msg.SetID(wrappedMsg.ID)

				msgCh <- msg
			case err := <-errCh:
				fmt.Println(err)
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgCh, errCh
}
