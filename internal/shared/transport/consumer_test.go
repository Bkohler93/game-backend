package transport

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/redis/go-redis/v9"
)

func TestStreamListener(t *testing.T) {
	numTests := 1000
	ctx := context.TODO()
	redisClient, err := redisutils.NewRedisMatchmakeClient(ctx)
	if err != nil {
		panic(err)
	}
	for j := 0; j < numTests; j++ {
		ctx = context.Background()
		s1 := "test1"
		listener := NewRedisStreamListener(ctx, redisClient)
		//producer := NewRedisMessageProducer(redisClient, s1)
		//sendMsg := func(msg string) {
		//	raw, err := json.Marshal(msg)
		//	if err != nil {
		//		t.Fatalf("error marshalling json - %v\n", err)
		//	}
		//
		//	err = producer.Send(ctx, &message.Envelope{
		//		Type:     message.MatchmakingService,
		//		Payload:  raw,
		//		MetaData: nil,
		//	})
		//	if err != nil {
		//		t.Fatalf("could not send msg with err[%v]\n", err)
		//	}
		//}
		//consumer := listener.AddConsumer(s1)
		addCtx, cancel := context.WithTimeout(ctx, time.Second*1)
		_, err := listener.AddConsumer(addCtx, s1)
		cancel()
		if err != nil {
			t.Fatalf("failed to add consumer with err[%v]\n", err)
		}

		var wg sync.WaitGroup
		//wg.Add(1)
		//go func() {
		//	defer func() {
		//		wg.Done()
		//	}()
		//
		//	msgCh, errCh := consumer.StartReceiving(ctx)
		//
		//	errTimer := time.NewTimer(time.Second * 1)
		//
		//mainLoop:
		//	for {
		//		select {
		//		case msg := <-msgCh:
		//			fmt.Printf("received msg - %v\n", msg)
		//			break mainLoop
		//		case err := <-errCh:
		//			t.Errorf("received err - %v\n", err)
		//		case <-errTimer.C:
		//			t.Fatalf("test timed out\n")
		//		}
		//	}
		//}()
		numStreams := 2
		names := []string{s1}
		for i := 0; i < numStreams; i++ {
			name := fmt.Sprintf("buttcheeks%d", i)
			names = append(names, name)
			//timer := time.NewTimer(time.Millisecond * 100)
			//doneCh := make(chan struct{})
			//go func() {
			//	listener.AddConsumer(name)
			//	close(doneCh)
			//}()
			//select {
			//case <-timer.C:
			//	t.Fatalf("took to long to add stream[%s]\n", name)
			//case <-doneCh:
			//	break
			//}

			addCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			_, err := listener.AddConsumer(addCtx, name)
			cancel()
			if err != nil {
				t.Fatalf("failed to add consumer[%s] with err[%v]\n", name, err)
			}
			//if i == 1 {
			//	sendMsg("sup smelly")
			//}
		}
		wg.Wait()

		time.Sleep(time.Millisecond * 100)
		didStop := listener.StopListening()
		if !didStop {
			t.Fatalf("failed to stop listening to redis streams - cycle[%d]\n", j)
		}
		err = redisClient.Del(ctx, names...).Err()
		if err != nil {
			t.Fatalf("failed to delete keys with err - %v\n", err)
		}
		fmt.Printf("test cycle[%d] passed\n", j)
		//err = redisClient.Close()
		//if err != nil {
		//	t.Fatalf("failed to close redis client - %v\n", err)
		//}
	}
	t.Cleanup(func() {
		err := redisClient.Close()
		if err != nil {
			log.Printf("error closing main redis client - %v", err)
		}
	})
}

func TestConnectionRecyclingWithOps(t *testing.T) {
	numCycles := 10

	mainClient, err := redisutils.NewRedisMatchmakeClient(context.Background())
	if err != nil {
		t.Fatalf("Failed to create main client: %v", err)
	}

	t.Cleanup(func() {
		if cErr := mainClient.Close(); cErr != nil {
			t.Logf("Warning: Failed to close main client pool: %v", cErr)
		}
	})

	for i := 0; i < numCycles; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		conn := mainClient.Conn()

		pingErr := conn.Ping(ctx).Err()
		if pingErr != nil {
			t.Fatalf("Cycle %d: Ping failed on dedicated connection: %v", i, pingErr)
		}

		closeErr := conn.Close()
		if closeErr != nil {
			t.Fatalf("Cycle %d: Failed to close dedicated connection: %v", i, closeErr)
		}

		cancel()
	}

	t.Logf("Successfully cycled through %d connection closes without error.", numCycles)
}

func TestConnectionRecyclingWithXReadOps(t *testing.T) {
	numCycles := 10

	mainClient, err := redisutils.NewRedisMatchmakeClient(context.Background())
	if err != nil {
		t.Fatalf("Failed to create main client: %v", err)
	}

	t.Cleanup(func() {
		if cErr := mainClient.Close(); cErr != nil {
			t.Logf("Warning: Failed to close main client pool: %v", cErr)
		}
	})

	for i := 0; i < numCycles; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		conn := mainClient.Conn()

		conn.XRead(ctx, &redis.XReadArgs{
			Streams: []string{"poop", "0-0"},
			Count:   1,
			Block:   0,
			ID:      "",
		})
		pingErr := conn.Ping(ctx).Err()
		if pingErr != nil {
			t.Fatalf("Cycle %d: Ping failed on dedicated connection: %v", i, pingErr)
		}

		closeErr := conn.Close()
		if closeErr != nil {
			t.Fatalf("Cycle %d: Failed to close dedicated connection: %v", i, closeErr)
		}

		cancel()
	}

	t.Logf("Successfully cycled through %d connection closes without error.", numCycles)
}

func TestConnectionRecyclingNoOps(t *testing.T) {
	numCycles := 10

	mainClient, err := redisutils.NewRedisMatchmakeClient(context.Background())
	if err != nil {
		t.Fatalf("Failed to create main client: %v", err)
	}

	t.Cleanup(func() {
		if cErr := mainClient.Close(); cErr != nil {
			t.Logf("Warning: Failed to close main client pool: %v", cErr)
		}
	})

	for i := 0; i < numCycles; i++ {
		conn := mainClient.Conn()

		closeErr := conn.Close()
		if closeErr != nil {
			t.Fatalf("Cycle %d: Failed to close dedicated connection: %v", i, closeErr)
		}
	}

	t.Logf("Successfully cycled through %d connection closes without error.", numCycles)
}
