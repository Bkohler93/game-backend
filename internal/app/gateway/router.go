package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/bkohler93/game-backend/internal/app/game"
	"github.com/bkohler93/game-backend/internal/app/matchmake"
	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/message/metadata"
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
	handler := NewMessageHandler(ctx, r, client)

	eg.Go(func() error {
		return handler.StartListening(eCtx)
	})

	// eg.Go(func() error {
	// 	for {
	// 		select {
	// 		case <-eCtx.Done():
	// 			return nil
	// 		case clientMsg := <-handler.fromServerCh:
	// 			client.writeChan <- clientMsg
	// 		}
	// 	}
	// currentGamePhaseReceiveMethod := r.receiveMatchmaking
	// for currentGamePhaseReceiveMethod != nil {
	// 	var err error
	// 	currentGamePhaseReceiveMethod, err = currentGamePhaseReceiveMethod(eCtx, client)
	// 	if err != nil {
	// 		break
	// 	}
	// }
	// return nil
	// })

	eg.Go(func() error {
		for {
			select {
			case <-eCtx.Done():
				return nil
			case err := <-client.errChan:
				handler.HandleClientErr(err)
			case serverMsg := <-client.routeChan:
				handler.RouteServerMsg(eCtx, serverMsg)
				if err := serverMsg.AckFunc(ctx); err != nil {
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

type MessageHandler struct {
	router              *Router
	client              *Client
	state               ServerMessageHandlerState
	fromServerCh        chan *message.EnvelopeContext
	gameProducer        transport.DynamicMessageProducer
	matchmakeProducer   transport.MessageProducer
	matchmakingConsumer transport.MessageConsumer
	gameConsumer        transport.MessageConsumer
}

func NewMessageHandler(ctx context.Context, router *Router, client *Client) *MessageHandler {
	s := &MessageHandler{
		router:            router,
		client:            client,
		state:             MatchmakingServerMessageState,
		fromServerCh:      make(chan *message.EnvelopeContext),
		gameProducer:      router.transportFactory.GameplayServerMsgProducerBuilder(),
		matchmakeProducer: router.transportFactory.MatchmakingServerMsgProducerBuilder(),
	}
	matchmakingMsgConsumer, err := router.transportFactory.MatchmakingClientMsgConsumerBuilder(ctx, client.ID.String())
	if err != nil {
		log.Printf("error creating matchmake client message consumer - %v\n", err)
	}
	s.matchmakingConsumer = matchmakingMsgConsumer

	gameMsgConsumer, err := router.transportFactory.GameClientMsgConsumerBuilder(ctx, client.ID.String())
	if err != nil {
		log.Printf("error creating Game Client Message consumer - %v\n", err)
	}
	s.gameConsumer = gameMsgConsumer

	return s
}

func (s *MessageHandler) StartListening(ctx context.Context) error {
	matchmakeCh, matchmakeErrCh := s.matchmakingConsumer.StartReceiving(ctx)
	gameCh, gameErrCh := s.gameConsumer.StartReceiving(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-matchmakeErrCh:
			return err
		case err := <-gameErrCh:
			return err
		case envCtx := <-matchmakeCh:
			s.state = MatchmakingServerMessageState
			s.handleMatchmakingMessage(envCtx)
		case envCtx := <-gameCh:
			s.state = GameServerMessageState
			s.handleGameMessage(envCtx)
		}
	}
}

// handleMatchmakingMessage processes messages from the matchmaking server
func (s *MessageHandler) handleMatchmakingMessage(envCtx *message.EnvelopeContext) {
	if envCtx.Env.MetaData[metadata.TransitionTo] == metadata.Game {
		roomId := envCtx.Env.MetaData[metadata.RoomIDKey]
		if roomId == "" {
			log.Println("did not receive room id")
		}
		s.client.RoomID = uuidstring.ID(roomId)
		s.state = GameServerMessageState // Transition the state
	}
	s.client.writeChan <- envCtx
}

// handleGameMessage processes messages from the game server
func (s *MessageHandler) handleGameMessage(envCtx *message.EnvelopeContext) {
	nextState := envCtx.Env.MetaData[metadata.TransitionTo]
	if nextState == metadata.GameOver {
		s.state = MatchmakingServerMessageState // Transition back to matchmaking
	}
	s.client.writeChan <- envCtx
}

func (s *MessageHandler) HandleClientErr(err error) {
	switch s.state {
	case MatchmakingServerMessageState:
		msg := matchmake.NewExitMatchmakingMessage(s.client.ID)
		bytes, _ := json.Marshal(msg)
		err := s.matchmakeProducer.Send(context.Background(), &message.Envelope{
			Type:    message.MatchmakingService,
			Payload: bytes,
		})
		if err != nil {
			log.Printf("error sending matchmaking disconnect - %v", err)
		}
	case GameServerMessageState:
		var msg any
		switch err {
		case ErrClientClosedConnection:
			msg = message.NewClientQuitMessage(s.client.ID)
		case ErrClientDisconnected:
			msg = message.NewClientDisconnectMessage(s.client.ID)
		default:
			log.Printf("received unknown error message - %v", err)
		}
		bytes, _ := json.Marshal(msg)
		err := s.gameProducer.SendTo(context.Background(), s.client.RoomID, &message.Envelope{
			Type:    message.GameService,
			Payload: bytes,
		})
		if err != nil {
			log.Printf("error sending game server disconnect message - %v", err)
		}
	}
}

func (s *MessageHandler) RouteServerMsg(ctx context.Context, envCtx *message.EnvelopeContext) {
	switch message.ServiceType(envCtx.Env.Type) {
	case message.MatchmakingService:
		s.state = MatchmakingServerMessageState
		err := s.matchmakeProducer.Send(ctx, envCtx.Env)
		if err != nil {
			log.Println("failed to send matchmaking message - ", err)
		}
	case message.GameService:
		s.state = GameServerMessageState
		err := s.gameProducer.SendTo(ctx, s.client.RoomID, envCtx.Env)
		if err != nil {
			log.Println("failed to send gameplay message - ", err)
		}
	default:
		log.Printf("Gateway received invalid out-message type - %v - data %v\n", envCtx.Env.Type, envCtx.Env)
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
				roomId := env.Env.MetaData[metadata.RoomIDKey]
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
	if game.IsGameServerMessageType(messageType) {
		return message.GameService
	} else if matchmake.IsMatchmakeServerMessageType(messageType) {
		return message.MatchmakingService
	}
	return ""

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
