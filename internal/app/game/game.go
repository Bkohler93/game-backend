package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type Game struct {
	GameID           uuidstring.ID
	RoomID           uuidstring.ID
	Players          []uuidstring.ID
	transportFactory *TransportFactory
}

func NewGame(roomID uuidstring.ID, tf *TransportFactory, playerIds []uuidstring.ID) *Game {
	return &Game{
		GameID:           uuidstring.NewID(),
		RoomID:           roomID,
		Players:          playerIds,
		transportFactory: tf,
	}
}

func (g *Game) Start(ctx context.Context) error {
	//create stream consumers/producers
	//TODO
	// g.
	// 1. send all players their first messages
	producer := g.transportFactory.GameClientMsgProducerBuilder()
	for _, playerId := range g.Players {
		msg := NewStartSetupMessage(
			[]string{"tit", "tat", "pup"},
			[]string{"tits", "ball", "hump"},
			[]string{"touch", "brain", "plane"},
		)
		bytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("failed to marshal msg into json - %v\n", err)
			continue
		}
		env := &message.Envelope{
			Type:    "",
			Payload: bytes,
		}

		producer.SendTo(ctx, playerId, env)
	}

	fmt.Println("ready to start setup phase")
	// 2. do the setup phase
	// 3. do the gameplay phase
	// 4. do the finish game phase
	return nil
}
