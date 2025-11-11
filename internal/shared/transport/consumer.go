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
	AddStreamCh     chan *streamReq
	RemoveStreamCh  chan string
	ctx             context.Context
	conn            *redis.Conn
	rdb             *redis.Client
	connID          int64
	streams         []string
	lastReceivedIDs []string
	streamsMu       sync.RWMutex
	msgChannels     map[string]chan *message.EnvelopeContext
	errChannels     map[string]chan error
	stopListeningCh chan any
	stoppedCh       chan any
}

func NewRedisStreamListener(ctx context.Context, rdb *redis.Client) *RedisStreamListener {
	conn := rdb.Conn()
	//connID := conn.ClientID(ctx).Val()

	listener := &RedisStreamListener{
		ctx:  ctx,
		rdb:  rdb,
		conn: conn,
		//connID:          connID,
		AddStreamCh:     make(chan *streamReq),
		RemoveStreamCh:  make(chan string),
		streams:         []string{},
		streamsMu:       sync.RWMutex{},
		lastReceivedIDs: []string{},
		msgChannels:     make(map[string]chan *message.EnvelopeContext),
		errChannels:     make(map[string]chan error),
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
		streams := append([]string(nil), l.streams...)
		ids := append([]string(nil), l.lastReceivedIDs...)
		l.streamsMu.RUnlock()
		doneCh := make(chan struct{})
		var cancel context.CancelFunc
		var ctx context.Context
		var stop func() bool

		wg := sync.WaitGroup{}

		// Use a new context for each XRead call to avoid re-using a cancelled one.
		ctx, cancel = context.WithCancel(l.ctx)

		if len(streams) > 0 {
			wg.Add(1)
			go func(lc int) {
				defer func() {
					close(doneCh)
					wg.Done()
				}()
				xReadConn := l.conn
				ID := xReadConn.ClientID(ctx).Val()
				//currentID := l.conn.ClientID(ctx).Val()
				stop = context.AfterFunc(ctx, func() {
					fmt.Printf("cancelling XRead on conn[%d]\n", ID)
					//unblockCtx, cancelUnblock := context.WithTimeout(context.Background(), time.Millisecond*100)
					//defer cancelUnblock()

					//err := l.rdb.ClientUnblock(unblockCtx, currentID).Err()
					//err := l.rdb.ClientUnblock(l.ctx, currentID).Err()
					//err := l.rdb.ClientUnblock(l.ctx, currentID).Err() //use the listener's active context to Unblock redis
					err := xReadConn.Close()
					if err != nil {
						log.Printf("error unblocking conn[%d] - %v - loopCount[%d]\n", l.connID, err, lc)
					}
					fmt.Printf("cancelled XRead\n")
				})
				fmt.Printf("listening to streams on conn[%d] [%v]\n", ID, streams)
				streamResults, err := xReadConn.XRead(ctx, &redis.XReadArgs{
					Streams: append(streams, ids...),
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
			}(loopCount)
		}
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

		case newStreamReq := <-l.AddStreamCh:
			if len(streams) > 0 {
				cancel()
				<-doneCh
				l.conn = l.rdb.Conn()
			}
			//select {
			//case <-doneCh:
			//case <-time.After(time.Second * 1):
			//	log.Printf("FATAL: Timed out waiting for XRead routine to close doneCh during stream addition.")
			//}
			//<-doneCh
			//wg.Wait()
			l.streamsMu.Lock()
			l.streams = append(l.streams, newStreamReq.stream)
			l.lastReceivedIDs = append(l.lastReceivedIDs, "0-0")
			l.streamsMu.Unlock()
			close(newStreamReq.reqDone)
			fmt.Printf("successfully added stream[%s]\n", newStreamReq.stream)
			continue
		case removeStream := <-l.RemoveStreamCh:
			cancel()
			//wg.Wait()
			<-doneCh
			l.streamsMu.Lock()
			idx := slices.Index(l.streams, removeStream)
			if idx == -1 {
				log.Printf("trying to remove non-existent stream - %s\n", removeStream)
				l.streamsMu.Unlock()
				continue
			}
			l.streams = append(l.streams[:idx], l.streams[idx+1:]...)
			l.lastReceivedIDs = append(l.lastReceivedIDs[:idx], l.lastReceivedIDs[idx+1:]...)
			delete(l.msgChannels, removeStream)
			delete(l.errChannels, removeStream)
			l.streamsMu.Unlock()
		}
	}
}

// AddConsumer adds a new stream to the stream listener
func (l *RedisStreamListener) AddConsumer(ctx context.Context, stream string) (*RedisMessageConsumer, error) {
	l.streamsMu.Lock()
	l.msgChannels[stream] = make(chan *message.EnvelopeContext, 100)
	l.errChannels[stream] = make(chan error, 10)
	l.streamsMu.Unlock()
	req := newStreamReq(stream)
	l.AddStreamCh <- req

	select {
	case <-req.reqDone:
		return &RedisMessageConsumer{
			streamListener: l,
			stream:         stream,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// RemoveConsumer removes a stream from the stream listener
func (l *RedisStreamListener) RemoveConsumer(stream string) {
	l.RemoveStreamCh <- stream
}

type RedisMessageConsumer struct {
	streamListener *RedisStreamListener
	stream         string
}

func (r RedisMessageConsumer) StartReceiving(ctx context.Context) (<-chan *message.EnvelopeContext, <-chan error) {
	msgCh := r.streamListener.msgChannels[r.stream]
	errCh := r.streamListener.errChannels[r.stream]
	return msgCh, errCh
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
