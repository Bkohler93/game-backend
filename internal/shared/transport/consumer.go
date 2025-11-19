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

type streamReq struct {
	stream  string
	reqDone chan any
}

func newStreamReq(stream string) *streamReq {
	return &streamReq{
		stream:  stream,
		reqDone: make(chan any),
	}
}

type RedisStreamListener struct {
	ctx             context.Context
	conn            *redis.Conn
	rdb             *redis.Client //used to cancel running XRead
	connID          int64
	streams         []string
	lastReceivedIDs []string
	streamsMu       sync.RWMutex
	msgChannels     map[string]chan *message.EnvelopeContext
	msgMu           sync.Mutex
	stopListeningCh chan any
	stoppedCh       chan any
}

func NewRedisStreamListener(ctx context.Context, rdb *redis.Client, streams []string) *RedisStreamListener {
	conn := rdb.Conn()
	connID := conn.ClientID(ctx).Val()
	var streamIDs []string
	for range streams {
		streamIDs = append(streamIDs, "0-0")
	}

	listener := &RedisStreamListener{
		ctx:    ctx,
		rdb:    rdb,
		conn:   conn,
		connID: connID,
		//AddStreamCh:     make(chan *streamReq),
		//RemoveStreamCh:  make(chan string),
		streams:         streams,
		streamsMu:       sync.RWMutex{},
		lastReceivedIDs: streamIDs,
		msgChannels:     make(map[string]chan *message.EnvelopeContext),
		msgMu:           sync.Mutex{},
		stopListeningCh: make(chan any),
		stoppedCh:       make(chan any),
	}

	go listener.runLoop()

	return listener
}

func (l *RedisStreamListener) StopListening() bool {
	close(l.stopListeningCh)

	select {
	case <-time.After(time.Second * 5):
		return false
	case <-l.stoppedCh:
		return true
	}
}

func (l *RedisStreamListener) runLoop() {
	loopCount := 0
	for {
		loopCount++
		l.streamsMu.RLock()
		ids := append([]string(nil), l.lastReceivedIDs...)
		l.streamsMu.RUnlock()
		doneCh := make(chan struct{})
		var cancel context.CancelFunc
		var ctx context.Context
		var stop func() bool

		wg := sync.WaitGroup{}

		// Use a new context for each XRead call to avoid re-using a cancelled one.
		ctx, cancel = context.WithCancel(l.ctx)

		wg.Add(1)
		go func(lc int) {
			defer func() {
				close(doneCh)
				wg.Done()
			}()
			//currentID := l.conn.ClientID(ctx).Val()
			stop = context.AfterFunc(ctx, func() {
				fmt.Printf("cancelling XRead on conn[%d]\n", l.connID)
				unblockCtx, cancelUnblock := context.WithTimeout(context.Background(), time.Millisecond*100)
				defer cancelUnblock()

				err := l.rdb.ClientUnblock(unblockCtx, l.connID).Err()
				if err != nil {
					log.Printf("error unblocking conn[%d] - %v - loopCount[%d]\n", l.connID, err, lc)
				}
				fmt.Printf("cancelled XRead\n")
			})
			fmt.Printf("listening to streams on conn[%d] [%v]\n", l.connID, l.streams)
			streamResults, err := l.conn.XRead(ctx, &redis.XReadArgs{
				Streams: append(l.streams, ids...),
				Count:   1,
				Block:   0,
			}).Result()
			fmt.Printf("stopped listening to streams\n")
			if errors.Is(err, redis.Nil) || errors.Is(err, redis.ErrClosed) {
				return
			}
			if err != nil {
				log.Printf("XRead error - %v\n", err)
				return
			}
			for _, streamResult := range streamResults {
				currentStream := streamResult.Stream
				msg := streamResult.Messages[0]
				//TODO Send MSG ID to Deduplicator to check if msg has already been processed.
				//TODO this will be implemented once there are multiple workers running run Loop.
				payload, ok := msg.Values["payload"].(string)
				if !ok {
					fmt.Printf("missing payload field for msg[%s]\n", msg.ID)
					continue
				}

				var env *message.Envelope
				if err := json.Unmarshal([]byte(payload), &env); err != nil {
					fmt.Printf("unmarshal failed for msg[%s] with err[%v]", msg.ID, err)
					continue
				}

				//TODO should Deduplicator update the lastReceivedIds?
				l.streamsMu.Lock()
				idx := slices.Index(l.streams, currentStream)
				if idx >= 0 {
					l.lastReceivedIDs[idx] = msg.ID
				}
				l.streamsMu.Unlock()

				var consumerID string
				consumerID = string(env.MetaData[metadata.DestinationID])

				l.msgMu.Lock()
				l.msgChannels[consumerID] <- &message.EnvelopeContext{
					Env: env,
					AckFunc: func(ctx context.Context) error {
						return nil
					},
				}
				l.msgMu.Unlock()
			}
		}(loopCount)
		select {
		case <-l.stopListeningCh:
			fmt.Printf("stopping runLoop\n")
			cancel()
			time.Sleep(time.Millisecond * 50)
			<-doneCh
			fmt.Printf("stopped runLoop\n")
			close(l.stoppedCh)
			return
		case <-doneCh:
			stop()
			cancel() // Cancel the context once XRead returns
		}
	}
}

// AddConsumer adds a new consumer to the consumerID listener
func (l *RedisStreamListener) AddConsumer(consumerID string) (*RedisMessageConsumer, error) {
	l.msgMu.Lock()
	l.msgChannels[consumerID] = make(chan *message.EnvelopeContext, 100)
	l.msgMu.Unlock()

	return &RedisMessageConsumer{
		streamListener: l,
		consumerID:     consumerID,
	}, nil
}

// RemoveConsumer removes a consumer from the consumerID listener
func (l *RedisStreamListener) RemoveConsumer(consumerID string) {
	l.msgMu.Lock()
	delete(l.msgChannels, consumerID)
	l.msgMu.Unlock()
}

type RedisMessageConsumer struct {
	streamListener *RedisStreamListener
	consumerID     string
}

func (r RedisMessageConsumer) StartReceiving(ctx context.Context) (<-chan *message.EnvelopeContext, <-chan error) {
	msgCh := r.streamListener.msgChannels[r.consumerID]
	return msgCh, nil
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
					errCh <- fmt.Errorf("error reading from RedisMessageGroupConsumer consumerID - %v", err)
					return
				}

				data, ok := streamResults[0].Messages[0].Values["payload"].(string)
				if !ok {
					log.Printf("error trying to structure the consumerID reply - %v", err)
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
