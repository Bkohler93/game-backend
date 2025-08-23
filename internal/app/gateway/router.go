package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/bkohler93/game-backend/internal/app/game"
	"github.com/bkohler93/game-backend/internal/app/matchmake"
	"github.com/bkohler93/game-backend/internal/shared/constants/metadata"
	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"golang.org/x/sync/errgroup"
)

type RouteFunc func(context.Context, *Client) (RouteFunc, error)

type Router struct {
	transportFactory *TransportFactory
}

func (r *Router) RouteClientTraffic(ctx context.Context, client *Client) {
	eg, eCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		currentGamePhaseReceiveMethod := r.receiveMatchmaking
		for currentGamePhaseReceiveMethod != nil {
			var err error
			currentGamePhaseReceiveMethod, err = currentGamePhaseReceiveMethod(eCtx, client)
			if err != nil {
				break
			}
		}
		return nil
	})

	eg.Go(func() error {
		handler := NewServerMessageHandler(r, client)
		for {
			select {
			case <-eCtx.Done():
				handler.HandleCtxDone()
				return nil
			case outMsg := <-client.routeChan:
				handler.HandleOutMsg(eCtx, outMsg)
				if err := outMsg.AckFunc(ctx); err != nil {
					log.Println("trouble Acknowledging message that was just sent -", err)
				}
			}
		}
	})

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Println("RouteClientTraffic ended with an error - ", err)
	}
}

type ServerMessageHandlerState int

const (
	MatchmakingServerMessageState ServerMessageHandlerState = iota
	GameServerMessageState
)

type ServerMessageHandler struct {
	router            *Router
	client            *Client
	state             ServerMessageHandlerState
	gameProducer      transport.DynamicMessageProducer
	matchmakeProducer transport.MessageProducer
}

func NewServerMessageHandler(router *Router, client *Client) *ServerMessageHandler {
	s := &ServerMessageHandler{
		router:            router,
		client:            client,
		state:             MatchmakingServerMessageState,
		gameProducer:      router.transportFactory.GameplayServerMsgProducerBuilder(),
		matchmakeProducer: router.transportFactory.MatchmakingServerMsgProducerBuilder(),
	}
	return s
}

func (s *ServerMessageHandler) HandleCtxDone() {
	if s.state == MatchmakingServerMessageState {
		msg := matchmake.NewExitMatchmakingMessage(s.client.ID)
		bytes, _ := json.Marshal(msg)
		err := s.matchmakeProducer.Send(context.Background(), &message.Envelope{
			Type:    message.MatchmakingService,
			Payload: bytes,
		})
		if err != nil {
			log.Printf("error sending matchmaking disconnect - %v", err)
		}
	} else {
		// TODO: implement game server disconnect
	}
}

func (s *ServerMessageHandler) HandleOutMsg(ctx context.Context, envCtx *message.EnvelopeContext) {
	switch message.ServiceType(envCtx.Env.Type) {
	case message.MatchmakingService:
		if s.state != MatchmakingServerMessageState {
			log.Println("Warning: Received matchmaking message in gameplay state")
		}
		err := s.matchmakeProducer.Send(ctx, envCtx.Env)
		if err != nil {
			log.Println("failed to send matchmaking message - ", err)
		}
	case message.GameService:
		if s.state != GameServerMessageState {
			s.state = GameServerMessageState
		}
		err := s.gameProducer.SendTo(ctx, s.client.RoomID, envCtx.Env)
		if err != nil {
			log.Println("failed to send gameplay message - ", err)
		}
	default:
		log.Printf("received invalid message type - %v", envCtx.Env.Type)
	}
}

func (r *Router) receiveMatchmaking(ctx context.Context, client *Client) (RouteFunc, error) {
	msgSource, err := r.transportFactory.MatchmakingClientMsgConsumerBuilder(ctx, client.ID.String())
	if err != nil {
		log.Printf("error creating matchmake client message consumer - %v\n", err)
	}
	envCh, errCh := msgSource.StartReceiving(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			//TODO err may be recoverable, may not need to return an err here
			log.Println("receiveMatchmaking MessageGroupConsumer received an error -", err)
			return nil, err
		case env := <-envCh:
			if env.Env.MetaData[metadata.TransitionTo] == metadata.Game {
				roomId := env.Env.MetaData[metadata.RoomID]
				if roomId == "" {
					log.Println("did not receive room id")
				}
				client.RoomID = uuidstring.ID(roomId)
				client.writeChan <- env
				return r.receiveGame, nil
			}
			client.writeChan <- env
		}
	}
}

func (r *Router) receiveGame(ctx context.Context, client *Client) (RouteFunc, error) {
	msgSource, err := r.transportFactory.GameClientMsgConsumerBuilder(ctx, client.ID.String())
	if err != nil {
		log.Printf("error creating Game Client Message consumer - %v\n", err)
	}
	msgCh, errCh := msgSource.StartReceiving(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			//TODO err may be recoverable, may not need to return an err here
			log.Println("receiveGame MessageGroupConsumer received an error -", err)
			return nil, err
		case msg := <-msgCh:
			nextState := msg.Env.MetaData[metadata.TransitionTo]
			switch nextState {
			case metadata.Remain:
				client.writeChan <- msg
			case metadata.GameOver:
				client.writeChan <- msg
				return nil, nil
			default:
				log.Println("unknown game state to start receiving messages for -", nextState)
			}
		}
	}
}

func getServiceType(messageType string) message.ServiceType {
	switch messageType {
	case string(matchmake.RequestMatchmaking), string(matchmake.ExitMatchmaking):
		return message.MatchmakingService
	case string(game.ConfirmMatch), string(game.GameFirstGuess), string(game.GameGuess), string(game.SetupFinalize), string(game.SetupPlacementAttempt), string(game.SetupUndo):
		return message.GameService
	default:
		return "" // or return an error/unknown service type
	}
}

//func NewClientTransportBusFactory(rdb *redis.Client, matchmakingClientMsgConsumerBuilder transport.MessageGroupConsumerBuilderFunc, matchmakingServerMessageProducerBuilder transport.MessageProducerBuilderFunc) *TransportFactory {
//	return &TransportFactory{
//		rdb,
//		matchmakingClientMsgConsumerBuilder,
//		matchmakingServerMessageProducerBuilder,
//	}
//}

//type TransportBus struct {
//	bus *transport.Bus
//}

//func (f *TransportFactory) NewTransportBus(clientId uuidstring.ID) *TransportBus {
//	b := &TransportBus{
//		bus: &transport.Bus{},
//	}
//	clientMessageConsumer := f.matchmakingClientMsgConsumerBuilder(clientId.String())
//	serverMessageProducer := f.matchmakingServerMsgProducerBuilder()
//
//	b.bus.AddMessageGroupConsumer(ClientMessageConsumer, clientMessageConsumer)
//	b.bus.AddMessageProducer(ServerMessageProducer, serverMessageProducer)
//	return b
//}

//func (f *TransportFactory) NewMatchmakingMsgConsumer(clientId uuidstring.ID) transport.MessageGroupConsumer {
//	return f.matchmakingClientMsgConsumerBuilder(clientId.String())
//}

//func (f *TransportFactory) NewSetupMsgConsumer(clientId uuidstring.ID) transport.MessageGroupConsumer {
//	return f.setupClientMsgConsumerBuilder(clientId.String())
//}

//func (b *TransportBus) StartReceivingMatchmakingClientMessages(ctx context.Context) (<-chan transport.AckableMessage, <-chan error) {
//	return b.bus.StartReceiving(ctx, ClientMessageConsumer)
//}
//
//func (b *TransportBus) AckMatchmakingMsg(ctx context.Context, id string) error {
//	return b.bus.AckMessage(ctx, ClientMessageConsumer, id)
//}
//
//func (b *TransportBus) SendMatchmakingServerMessage(ctx context.Context, payload json.RawMessage) error {
//	return b.bus.Send(ctx, ServerMessageProducer, payload)
//}
