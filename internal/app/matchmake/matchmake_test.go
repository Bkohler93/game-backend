package matchmake

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
)

func TestRedisMatchmakingTransportBus(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	startup := func(t *testing.T) (bus *TransportBus, flush func()) {
		client, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		serverMessageConsumer, err := NewRedisMatchmakingServerMessageConsumer(ctx, client, rediskeys.MatchmakingServerMessageStream)
		if err != nil {
			t.Errorf("unpexected error creating server message consumer")
		}
		clientMessageProducer := NewRedisClientMessageProducer(client)
		matchmakeWorkerNotifier := NewRedisWorkerNotifierBroadcastProducer(client)
		matchmakeWorkerNotifyReceiver := NewRedisWorkerNotifierListener(client)

		bus = NewBus(serverMessageConsumer, clientMessageProducer, matchmakeWorkerNotifier, matchmakeWorkerNotifyReceiver)
		flush = func() {
			cancel()
			err := client.FlushDB(context.Background()).Err()
			if err != nil {
				panic(err)
			}
		}
		return
	}

	t.Run("test worker notifier", func(t *testing.T) {
		bus, flush := startup(t)
		defer flush()
		wg := sync.WaitGroup{}

		msgCh, errCh := bus.ListenForMatchmakeWorkerNotifications(ctx)
		go func() {
			wg.Add(1)
			defer wg.Done()
			timer := time.NewTimer(time.Second * 1)
			for {
				select {
				case msg := <-msgCh:
					if msg.Type != "" {
						t.Errorf("expected empty message got - %s", msg)
					}
					timer.Stop()
					return
				case err := <-errCh:
					t.Errorf("unexpected error from worker notify channel - %v", err)
					timer.Stop()
					return
				case <-timer.C:
					t.Error("have not received notification in time")
					return
				}
			}
		}()

		<-time.NewTimer(time.Millisecond * 20).C //allow some time for subscription to finish

		err := bus.NotifyMatchmakeWorkers(ctx)
		if err != nil {
			t.Errorf("unexpected error broadcasting over matchmake worker notifier pubsub - %v", err)
		}

		wg.Wait()
	})
}
