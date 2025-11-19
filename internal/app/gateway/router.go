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

func (r *Router) UnrouteClientTraffic(c *Client) {
	r.transportFactory.GameClientMsgConsumerDestroyer(c.ID)
	r.transportFactory.MatchmakingClientMsgConsumerDestroyer(c.ID)
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
	gameProducer        transport.MessageProducer
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

func getServiceType(messageType string) message.ServiceType {
	if game.IsGameServerMessageType(messageType) {
		return message.GameService
	} else if matchmake.IsMatchmakeServerMessageType(messageType) {
		return message.MatchmakingService
	}
	return ""
}
