package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/message/metadata"
	"github.com/bkohler93/game-backend/internal/shared/utils/files"
	"github.com/redis/go-redis/v9"
)

//type TransportMessage interface {
//	Ack(context.Context) error
//	GetPayload() any
//	GetMetadata() map[string]interface{}
//}

//type AckableMessage struct {
//	AckFunc  func(context.Context) error
//	Payload  any
//	Metadata map[string]interface{}
//}

//func (a AckableMessage) Ack(ctx context.Context) error {
//	if a.AckFunc != nil {
//		return a.AckFunc(ctx)
//	}
//	return nil
//}
//
//func (a AckableMessage) GetPayload() any {
//	return a.Payload
//}
//
//func (a AckableMessage) GetMetadata() map[string]interface{} {
//	return a.Metadata
//}

//type Message struct {
//	Payload  any
//	Metadata map[string]interface{}
//}

//func (m Message) Ack(ctx context.Context) error {
//	return nil
//}
//
//func (m Message) GetPayload() any {
//	return m.Payload
//}
//
//func (m Message) GetMetadata() map[string]interface{} {
//	return m.Metadata
//}

var (
	ErrScriptNotFound = errors.New("script not found")
)

type MessageConsumerType string
type MessageConsumer interface {
	StartReceiving(ctx context.Context) (<-chan *message.EnvelopeContext, <-chan error)
}
type MessageConsumerBuilderFunc func(ctx context.Context, streamSuffix string) (MessageConsumer, error)

type RedisStreamListener struct {
	restartCh       chan struct{}
	ctx             context.Context
	rdb             *redis.Client
	streams         []string
	lastReceivedIDs []string
	streamsMu       sync.RWMutex
	msgChannels     map[string]chan *message.EnvelopeContext
	errChannels     map[string]chan error
}

func NewRedisStreamListener(ctx context.Context, rdb *redis.Client) *RedisStreamListener {
	listener := &RedisStreamListener{
		ctx:             ctx,
		rdb:             rdb,
		restartCh:       make(chan struct{}, 1),
		streams:         []string{},
		lastReceivedIDs: []string{},
		msgChannels:     make(map[string]chan *message.EnvelopeContext),
		errChannels:     make(map[string]chan error),
	}

	go listener.runLoop()

	return listener
}

func (l *RedisStreamListener) runLoop() {
	for {
		l.streamsMu.RLock()
		streams := append([]string(nil), l.streams...)
		ids := append([]string(nil), l.lastReceivedIDs...)
		l.streamsMu.RUnlock()

		if len(streams) == 0 {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		// Use a new context for each XRead call to avoid re-using a cancelled one.
		ctx, cancel := context.WithCancel(l.ctx)

		// Asynchronously call XRead so it can be unblocked by the select statement.
		ch := make(chan struct {
			results []redis.XStream
			err     error
		})
		go func() {
			streamResults, err := l.rdb.XRead(ctx, &redis.XReadArgs{
				Streams: append(streams, ids...),
				Count:   1,
				Block:   0,
			}).Result()
			ch <- struct {
				results []redis.XStream
				err     error
			}{streamResults, err}
		}()

		select {
		case result := <-ch:
			cancel() // Cancel the context once XRead returns
			if result.err != nil {
				if errors.Is(result.err, context.Canceled) {
					// This can happen if the signal was received and the context was canceled.
					continue
				}
				log.Printf("error reading from streams: %v", result.err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Process results (your existing logic)
			for _, streamResult := range result.results {
				currentStream := streamResult.Stream
				msg := streamResult.Messages[0]

				payload, ok := msg.Values["payload"].(string)
				if !ok {
					l.errChannels[currentStream] <- fmt.Errorf("missing payload field")
					continue
				}

				var env *message.Envelope
				if err := json.Unmarshal([]byte(payload), &env); err != nil {
					l.errChannels[currentStream] <- fmt.Errorf("unmarshal failed: %w", err)
					continue
				}

				l.streamsMu.Lock()
				idx := slices.Index(l.streams, currentStream)
				if idx >= 0 {
					l.lastReceivedIDs[idx] = msg.ID
				}
				l.streamsMu.Unlock()

				l.msgChannels[currentStream] <- &message.EnvelopeContext{
					Env: env,
					AckFunc: func(ctx context.Context) error {
						return nil
					},
				}
			}
		case <-l.restartCh:
			cancel() // The restart signal was received, unblock the XRead call.
			continue
		}
	}
}

// func (l *RedisStreamListener) runLoop() {
// 	for {
// 		l.streamsMu.RLock()
// 		streams := append([]string(nil), l.streams...)
// 		ids := append([]string(nil), l.lastReceivedIDs...)
// 		l.streamsMu.RUnlock()

// 		if len(streams) == 0 {
// 			time.Sleep(50 * time.Millisecond)
// 			continue
// 		}
// 		fmt.Printf("listening for streams %#v\n", streams)
// 		streamResults, err := l.rdb.XRead(l.ctx, &redis.XReadArgs{
// 			Streams: append(streams, ids...),
// 			Count:   1,
// 			Block:   0,
// 		}).Result()

// 		if err != nil {
// 			if errors.Is(err, context.Canceled) {
// 				l.ctx, l.cancelCtx = context.WithCancel(context.Background())
// 				fmt.Printf("stream listsener runLoop context cancelled\n")
// 				// XRead unblocked due to cancel, just restart loop
// 				continue
// 			}
// 			log.Printf("error reading from streams: %v", err)
// 			time.Sleep(100 * time.Millisecond)
// 			continue
// 		}
// 		for _, streamResult := range streamResults {
// 			currentStream := streamResult.Stream
// 			msg := streamResult.Messages[0]

// 			payload, ok := msg.Values["payload"].(string)
// 			if !ok {
// 				l.errChannels[currentStream] <- fmt.Errorf("missing payload field")
// 				continue
// 			}

// 			var env *message.Envelope
// 			if err := json.Unmarshal([]byte(payload), &env); err != nil {
// 				l.errChannels[currentStream] <- fmt.Errorf("unmarshal failed: %w", err)
// 				continue
// 			}

// 			l.streamsMu.Lock()
// 			idx := slices.Index(l.streams, currentStream)
// 			if idx >= 0 {
// 				l.lastReceivedIDs[idx] = msg.ID
// 			}
// 			l.streamsMu.Unlock()

// 			l.msgChannels[currentStream] <- &message.EnvelopeContext{
// 				Env: env,
// 				AckFunc: func(ctx context.Context) error {
// 					return nil
// 				},
// 			}
// 		}
// 	}
// }

// AddConsumer a stream to the stream listener and restarts the XRead loop  to include the newly added stream
func (l *RedisStreamListener) AddConsumer(stream string) *RedisMessageConsumer {
	l.streamsMu.Lock()
	l.streams = append(l.streams, stream)
	l.lastReceivedIDs = append(l.lastReceivedIDs, "0-0")
	l.msgChannels[stream] = make(chan *message.EnvelopeContext, 100)
	l.errChannels[stream] = make(chan error, 10)
	l.streamsMu.Unlock()

	select {
	case l.restartCh <- struct{}{}:
	default:
		// Channel is full, a restart is already pending.
		// No need to block or send another signal.
	}

	return &RedisMessageConsumer{
		streamListener: l,
		stream:         stream,
	}
}

type RedisMessageConsumer struct {
	streamListener *RedisStreamListener
	stream         string
}

// type RedisStreamListener struct {
// 	ctx             context.Context
// 	cancelCtx       context.CancelFunc
// 	rdb             *redis.Client
// 	streams         []string
// 	streamsMu       sync.RWMutex
// 	lastReceivedIDs []string
// 	msgChannels     map[string]chan *message.EnvelopeContext
// 	errChannels     map[string]chan error
// }

// func NewRedisStreamListener(ctx context.Context, rdb *redis.Client) *RedisStreamListener {
// 	listenerCtx, cancelCtx := context.WithCancel(ctx)
// 	listener := &RedisStreamListener{
// 		ctx:             listenerCtx,
// 		cancelCtx:       cancelCtx,
// 		rdb:             rdb,
// 		streams:         []string{},
// 		streamsMu:       sync.RWMutex{},
// 		lastReceivedIDs: []string{},
// 		msgChannels:     make(map[string]chan *message.EnvelopeContext),
// 		errChannels:     make(map[string]chan error),
// 	}

// 	go func() {
// 		for {
// 			fmt.Println("listening for message from XRead")
// 			select {
// 			case <-listenerCtx.Done():
// 				newCtx, newCancel := context.WithCancel(context.Background())
// 				listener.ctx = newCtx
// 				listener.cancelCtx = newCancel
// 				fmt.Println("streamListener context done")
// 				continue
// 			default:
// 				listener.streamsMu.RLock()
// 				streams := append([]string(nil), listener.streams...) // shallow copy
// 				ids := append([]string(nil), listener.lastReceivedIDs...)
// 				listener.streamsMu.RUnlock()

// 				fmt.Printf("listening for streams %#v\n", append(streams, ids...))
// 				streamResults, err := listener.rdb.XRead(ctx, &redis.XReadArgs{
// 					Streams: append(streams, ids...),
// 					Count:   1,
// 					Block:   0,
// 				}).Result()
// 				fmt.Println("received a message from XRead call")
// 				if err != nil {
// 					if errors.Is(err, context.Canceled) {
// 						return
// 					}
// 					fmt.Printf("error reading from RedisStreamListener stream - %v", err)
// 					return
// 				}
// 				for _, streamResult := range streamResults {
// 					currentStream := streamResult.Stream
// 					data, ok := streamResult.Messages[0].Values["payload"].(string)
// 					if !ok {
// 						log.Printf("error trying to structure the stream reply - %v", err)
// 					}
// 					payload := []byte(data)
// 					messageId := streamResult.Messages[0].ID
// 					streamIdx := slices.Index(streams, streamResult.Stream)
// 					if streamIdx >= 0 {
// 						ids[streamIdx] = messageId
// 					}

// 					var env *message.Envelope
// 					err = json.Unmarshal(payload, &env)
// 					if err != nil {
// 						listener.errChannels[currentStream] <- fmt.Errorf("error trying to unmarshal envelope - %v", err)
// 						continue
// 					}

// 					listener.msgChannels[currentStream] <- &message.EnvelopeContext{
// 						Env: env,
// 						AckFunc: func(ctx context.Context) error {
// 							// streamListener.lastReceivedIDs[streamIdx] = messageId
// 							return nil
// 						},
// 					}
// 				}
// 			}
// 		}
// 	}()

// 	return listener
// }

// type RedisMessageConsumer struct {
// 	streamListener *RedisStreamListener
// 	// rdb            *redis.Client
// 	stream string
// }

// func NewRedisMessageConsumer(ctx context.Context, streamListener *RedisStreamListener, stream string) *RedisMessageConsumer {
// 	msgConsumer := &RedisMessageConsumer{
// 		streamListener: streamListener,
// 		stream:         stream,
// 	}
// 	streamListener.streamsMu.Lock()
// 	streamListener.streams = append(streamListener.streams, stream)
// 	streamListener.lastReceivedIDs = append(streamListener.lastReceivedIDs, "0-0")
// 	streamListener.streamsMu.Unlock()

// 	streamListener.msgChannels[stream] = make(chan *message.EnvelopeContext, 100)
// 	streamListener.errChannels[stream] = make(chan error)
// 	streamListener.cancelCtx()

// 	return msgConsumer
// }

func (r RedisMessageConsumer) StartReceiving(ctx context.Context) (<-chan *message.EnvelopeContext, <-chan error) {
	msgCh := r.streamListener.msgChannels[r.stream]
	errCh := r.streamListener.errChannels[r.stream]
	return msgCh, errCh
	// msgCh := make(chan *message.EnvelopeContext)
	// errCh := make(chan error, 1)
	// go func() {
	// 	defer close(msgCh)
	// 	defer close(errCh)
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		default:
	// 			streamResults, err := r.rdb.XRead(ctx, &redis.XReadArgs{
	// 				Streams: []string{r.stream},
	// 				Count:   1,
	// 				Block:   0,
	// 				ID:      r.lastReceivedID,
	// 			}).Result()
	// 			if err != nil {
	// 				if errors.Is(err, context.Canceled) {
	// 					return
	// 				}
	// 				errCh <- fmt.Errorf("error reading from RedisMessageConsumer stream - %v", err)
	// 				return
	// 			}
	// 			data, ok := streamResults[0].Messages[0].Values["payload"].(string)
	// 			if !ok {
	// 				log.Printf("error trying to structure the stream reply - %v", err)
	// 			}
	// 			payload := []byte(data)
	// 			r.lastReceivedID = streamResults[0].Messages[0].ID

	// 			var env *message.Envelope

	// 			err = json.Unmarshal(payload, &env)
	// 			if err != nil {
	// 				log.Printf("error trying to unmarshal envelope - %v", err)
	// 			}

	// 			msgCh <- &message.EnvelopeContext{
	// 				Env: env,
	// 				AckFunc: func(ctx context.Context) error {
	// 					return nil
	// 				},
	// 			}
	// 		}
	// 	}
	// }()
	// return msgCh, errCh
}

// type MessageGroupConsumerBuilderFunc = func(ctx context.Context, consumerId string) MessageConsumer
type MessageConsumerFactory interface {
	CreateConsumer(ctx context.Context, consumer string) (MessageConsumer, error)
}

type BroadcastConsumerType string
type BroadcastConsumer interface {
	Subscribe(ctx context.Context) (<-chan *message.Envelope, <-chan error)
}

type RedisMessageGroupConsumer struct {
	rdb           *redis.Client
	stream        string
	consumerGroup string
	consumer      string
	luaScripts    map[string]*redis.Script
}

var (
	consumerBlockDuration = time.Second * 5
)

func NewRedisMessageGroupConsumer(ctx context.Context, rdb *redis.Client, stream, consumerGroup, consumer string) (*RedisMessageGroupConsumer, error) {
	var r *RedisMessageGroupConsumer
	luaScripts := make(map[string]*redis.Script)
	atomicAckDelSrc, err := files.GetLuaScript(files.LuaCGroupAckDelMsg)
	if err != nil {
		return r, fmt.Errorf("error loading atomicAckDel lua script - %v", err)
	}
	luaScripts[files.LuaCGroupAckDelMsg] = redis.NewScript(atomicAckDelSrc)

	r = &RedisMessageGroupConsumer{
		rdb:           rdb,
		stream:        stream,
		consumerGroup: consumerGroup,
		consumer:      consumer,
		luaScripts:    luaScripts,
	}
	_, err = r.rdb.XGroupCreateMkStream(ctx, stream, consumerGroup, "$").Result()
	if err != nil && strings.Contains(err.Error(), "Group name already exists") {
		err = nil
	}
	return r, err
}

func (mc *RedisMessageGroupConsumer) StartReceiving(ctx context.Context) (<-chan *message.EnvelopeContext, <-chan error) {
	msgCh := make(chan *message.EnvelopeContext)
	errCh := make(chan error, 1)
	go func() {
		defer close(msgCh)
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				streamResults, err := mc.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    mc.consumerGroup,
					Consumer: mc.consumer,
					Streams:  []string{mc.stream, ">"},
					Count:    1,
					Block:    0,
				}).Result()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					errCh <- fmt.Errorf("error reading from RedisMessageGroupConsumer stream - %v", err)
					return
				}

				data, ok := streamResults[0].Messages[0].Values["payload"].(string)
				if !ok {
					log.Printf("error trying to structure the stream reply - %v", err)
				}
				payload := []byte(data)
				id := streamResults[0].Messages[0].ID

				var env *message.Envelope

				err = json.Unmarshal(payload, &env)
				if err != nil {
					log.Printf("error trying to unmarshal envelope - %v", err)
				}
				env.EnsureMetaData()
				env.MetaData[metadata.MsgIdKey] = metadata.MetaDataValue(id)

				msgCh <- &message.EnvelopeContext{
					Env: env,
					AckFunc: func(ctx context.Context) error {
						return nil
					},
				}
			}
		}
	}()
	return msgCh, errCh
}

func (mc *RedisMessageGroupConsumer) AckMessage(ctx context.Context, msgId string) error {
	if _, ok := mc.luaScripts[files.LuaCGroupAckDelMsg]; !ok {
		return ErrScriptNotFound
	}
	return mc.luaScripts[files.LuaCGroupAckDelMsg].Run(ctx, mc.rdb, []string{mc.stream}, mc.consumerGroup, msgId).Err()
}

type RedisBroadcastConsumer struct {
	rdb     *redis.Client
	channel string
}

func (r *RedisBroadcastConsumer) Subscribe(ctx context.Context) (<-chan *message.Envelope, <-chan error) {
	returnCh := make(chan *message.Envelope)
	errCh := make(chan error)

	receiveCh := r.rdb.Subscribe(ctx, r.channel).Channel()
	go func() {
		for {
			select {
			case msg := <-receiveCh:
				payload := msg.Payload
				var env *message.Envelope
				err := json.Unmarshal([]byte(payload), &env)
				if err != nil {
					log.Printf("failed to unmarshal broadcast into envelope - %v\n", err)
				}
				returnCh <- env
			case <-ctx.Done():
				errCh <- ctx.Err()
			}
		}
	}()

	return returnCh, errCh
}

func NewRedisBroadcastConsumer(rdb *redis.Client, channel string) *RedisBroadcastConsumer {
	r := &RedisBroadcastConsumer{rdb, channel}
	return r
}
