package gateway

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/bkohler93/game-backend/internal/shared/constants/metadata"
	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/transport"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

type RouteFunc func(context.Context, *Client) (RouteFunc, error)

type Router struct {
	transportFactory *TransportFactory
}

func (r *Router) RouteClientTraffic(ctx context.Context, client *Client) {
	currentGameReceiveRoute := r.receiveMatchmaking
	eg, eCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		for currentGameReceiveRoute != nil {
			var err error
			currentGameReceiveRoute, err = currentGameReceiveRoute(eCtx, client)
			if err != nil {
				break
			}
		}
		return nil
	})

	eg.Go(func() error {
		matchmakingMessageProducer := r.transportFactory.MatchmakingServerMsgProducerBuilder()
		var setupMessageProducer transport.DynamicMessageProducer
		var gameplayMessageProducer transport.DynamicMessageProducer
		for {
			select {
			case <-eCtx.Done():
				return nil
			case outMsg := <-client.routeChan:
				switch message.ServiceType(outMsg.Env.Type) {
				case message.MatchmakingService:
					err := matchmakingMessageProducer.Send(eCtx, outMsg.Env)
					if err != nil {
						log.Println("failed to send matchmaking message - ", err)
					}
				case message.SetupService:
					if setupMessageProducer == nil {
						setupMessageProducer = r.transportFactory.SetupServerMsgProducerBuilder()
					}
					err := setupMessageProducer.SendTo(ctx, client.RoomID, outMsg.Env)
					if err != nil {
						log.Println("failed to send setup message -", err)
					}
				case message.GameService:
					if gameplayMessageProducer == nil {
						gameplayMessageProducer = r.transportFactory.GameplayServerMsgProducerBuilder()
					}
					err := gameplayMessageProducer.SendTo(ctx, client.RoomID, outMsg.Env)
					if err != nil {
						log.Print("failed to send gameplay message -", err)
					}
				default:
					log.Printf("received invalid message type - %v\n", outMsg.Env.Type)
				}
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

func (r *Router) receiveMatchmaking(ctx context.Context, client *Client) (RouteFunc, error) {
	msgSource := r.transportFactory.MatchmakingClientMsgConsumerBuilder(client.ID.String())
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
			if env.Env.MetaData[metadata.NewGameState] == metadata.Setup {
				roomId := env.Env.MetaData[metadata.RoomID]
				if roomId == "" {
					log.Println("did not receive room id")
				}
				client.RoomID = uuidstring.ID(roomId)
				client.writeChan <- env
				return r.receiveSetup, nil
			}
			client.writeChan <- env
		}
	}
}

func (r *Router) receiveSetup(ctx context.Context, client *Client) (RouteFunc, error) {
	msgSource := r.transportFactory.SetupClientMsgConsumerBuilder(client.RoomID.String())
	msgCh, errCh := msgSource.StartReceiving(ctx)

	for {
		fmt.Println("waiting to receive setup messages =) THIS IS GOOD BE HAPPY")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			//TODO err may be recoverable, may not need to return an err here
			log.Println("receiveMatchmaking MessageGroupConsumer received an error -", err)
			return nil, err
		case msg := <-msgCh:
			nextState := msg.Env.MetaData[metadata.NewGameState]
			switch nextState {
			case metadata.Remain:
				client.writeChan <- msg
			case metadata.Play:
				client.writeChan <- msg
				return r.receiveGameplay, nil
			case metadata.Matchmake:
				log.Println("moving back to matchmaking - hopefully because a player left")
				client.writeChan <- msg
				return r.receiveMatchmaking, nil
			default:
				log.Println("unknown game state to start receiving messages for -", nextState)
			}
		}
	}
}

func (r *Router) receiveGameplay(ctx context.Context, client *Client) (RouteFunc, error) {
	msgSource := r.transportFactory.GameplayClientMsgConsumerBuilder(client.RoomID.String())
	msgCh, errCh := msgSource.StartReceiving(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			//TODO err may be recoverable, may not need to return an err here
			log.Println("receiveMatchmaking MessageGroupConsumer received an error -", err)
			return nil, err
		case msg := <-msgCh:
			nextState := msg.Env.MetaData[metadata.NewGameState]
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

const (
	ClientMessageConsumer transport.MessageConsumerType = "ClientMessageConsumer"
	ServerMessageProducer transport.MessageProducerType = "ServerMessageProducer"
)

type TransportFactory struct {
	rdb                                 *redis.Client
	MatchmakingClientMsgConsumerBuilder transport.MessageGroupConsumerBuilderFunc
	SetupClientMsgConsumerBuilder       transport.MessageConsumerBuilderFunc
	GameplayClientMsgConsumerBuilder    transport.MessageConsumerBuilderFunc

	MatchmakingServerMsgProducerBuilder transport.MessageProducerBuilderFunc
	SetupServerMsgProducerBuilder       transport.DynamicMessageProducerBuilderFunc
	GameplayServerMsgProducerBuilder    transport.DynamicMessageProducerBuilderFunc
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
