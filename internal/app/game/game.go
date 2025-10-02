package game

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/utils/files"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"golang.org/x/sync/errgroup"
)

const (
	GameTimeLimit      = 300 //seconds
	GameUpdateDuration = time.Second * 1
)

type Game struct {
	GameID          uuidstring.ID
	RoomID          uuidstring.ID
	currentState    GameState
	Players         []uuidstring.ID
	Gameboards      map[uuidstring.ID]*Gameboard
	FirstTurnWinner uuidstring.ID
	disconnectCh    chan message.Message
	emoteCh         chan message.Message
	actionCh        chan message.Message
	ctx             context.Context
	cancelCtx       context.CancelFunc
	sendToClient    func(ctx context.Context, clientId uuidstring.ID, msg GameClientMessage) error
}

func (g *Game) Shutdown() {
	g.cancelCtx()
}

func (g *Game) ProcessGameResult(id uuidstring.ID) {
	//TODO: store game result
	log.Printf("user[%s] won!\n", id)
}

func (g *Game) OtherPlayerId(id uuidstring.ID) uuidstring.ID {
	for _, playerId := range g.Players {
		if playerId != id {
			return playerId
		}
	}
	return ""
}

func NewGame(ctx context.Context, roomID uuidstring.ID, bus *GameTransportBus, playerIds []uuidstring.ID) *Game {
	gameCtx, cancelCtx := context.WithCancel(ctx)
	msgCh, errCh := bus.StartReceivingServerMessages(ctx)

	game := &Game{

		GameID:       uuidstring.NewID(),
		RoomID:       roomID,
		Players:      playerIds,
		Gameboards:   make(map[uuidstring.ID]*Gameboard),
		currentState: nil,
		disconnectCh: make(chan message.Message),
		emoteCh:      make(chan message.Message),
		actionCh:     make(chan message.Message),
		sendToClient: bus.SendToClient,
		ctx:          gameCtx,
		cancelCtx:    cancelCtx,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgCh:
				switch msg.Msg.GetDiscriminator() {
				case string(message.ClientDisconnect), string(message.ClientQuit):
					log.Printf("client disconnected with reason %s\n", msg.Msg.GetDiscriminator())
					game.disconnectCh <- msg.Msg
				case string(ClientEmote):
					game.emoteCh <- msg.Msg
				default:
					game.actionCh <- msg.Msg
				}
			case err := <-errCh:
				log.Printf("encountered an error from ServerMessage stream - %v\n", err)
			}
		}
	}()

	return game
}

func (g *Game) Start() error {
	var eg errgroup.Group

	eg.Go(func() error {
		for {
			select {
			case <-g.ctx.Done():
				return nil
			case msg := <-g.emoteCh:
				for _, playerId := range g.Players {
					log.Println("received emote ---", msg)
					outMsg := NewEmoteMessage()
					g.sendToClient(g.ctx, playerId, outMsg)
				}
			}
		}
	})

	eg.Go(func() error {
		for {
			select {
			case <-g.ctx.Done():
				return nil
			case msg := <-g.disconnectCh:
				g.currentState.HandleDisconnect(g.ctx, msg)
			}
		}
	})

	eg.Go(func() error {
		for {
			select {
			case <-g.ctx.Done():
				return nil
			case msg := <-g.actionCh:
				g.currentState.HandleAction(g.ctx, msg)
			}
		}
	})

	g.NextPhase(g.ctx)

	if err := eg.Wait(); err != nil {
		log.Println("Game exited due to error - ", err)
		return err
	} else {
		log.Println("Game exited normally")
	}
	return nil
}

func (g *Game) NextPhase(ctx context.Context) {
	switch g.currentState.(type) {
	case nil:
		g.currentState = NewConnecting(g)
	case *Connecting:
		g.currentState = &Setup{g, make(map[uuidstring.ID][][]string), make(map[uuidstring.ID]*WordPlacements), make(map[uuidstring.ID]*WordSelections)}
	case *Setup:
		g.currentState = NewTurnDecider(g)
	case *TurnDecider:
		g.currentState = NewGameplay(g)
	case *Gameplay:
		g.currentState = NewPostgame(g)
	}
	g.currentState.Enter(ctx)
}

type GameState interface {
	Enter(context.Context)
	HandleAction(context.Context, message.Message)
	HandleDisconnect(context.Context, message.Message)
}

type Connecting struct {
	g                *Game
	playersConnected map[uuidstring.ID]bool
}

func NewConnecting(g *Game) *Connecting {
	c := &Connecting{
		g:                g,
		playersConnected: make(map[uuidstring.ID]bool),
	}
	for _, playerId := range g.Players {
		c.playersConnected[playerId] = false
	}
	return c
}

func (c *Connecting) HandleAction(ctx context.Context, msg message.Message) {
	switch m := msg.(type) {
	case *ConfirmMatchMessage:
		c.playersConnected[m.UserId] = true
		if allPlayersConnected(&c.playersConnected) {
			c.g.NextPhase(ctx)
		}
	default:
		log.Println("received invalid action during connecting phase -", m)
	}
}

func (c *Connecting) Enter(ctx context.Context) {
}

func (c *Connecting) HandleDisconnect(ctx context.Context, msg message.Message) {
	switch m := msg.(type) {
	case *message.ClientQuitMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	case *message.ClientDisconnectMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	}
	c.g.Shutdown()
}

type Setup struct {
	g              *Game
	wordOptions    map[uuidstring.ID][][]string
	wordPlacements map[uuidstring.ID]*WordPlacements
	wordSelections map[uuidstring.ID]*WordSelections
}

func (c *Setup) Enter(ctx context.Context) {
	for _, playerId := range c.g.Players {
		threeLetterWords, err := files.GetThreeRandomWords(3)
		if err != nil {
			log.Printf("failed to retrieve 3 letter words - %v\n", err)
		}
		c.wordOptions[playerId] = append(c.wordOptions[playerId], threeLetterWords)
		fourLetterWords, err := files.GetThreeRandomWords(4)
		if err != nil {
			log.Printf("failed to retrieve 4 letter words - %v\n", err)
		}
		c.wordOptions[playerId] = append(c.wordOptions[playerId], fourLetterWords)
		fiveLetterWords, err := files.GetThreeRandomWords(5)
		if err != nil {
			log.Printf("failed to retrieve 5 letter words - %v\n", err)
		}
		c.wordOptions[playerId] = append(c.wordOptions[playerId], fiveLetterWords)
		msg := NewStartSetupMessage(
			threeLetterWords,
			fourLetterWords,
			fiveLetterWords,
		)

		c.g.sendToClient(ctx, playerId, msg)
	}
}

// HandleAction implements the GameState interface for Setup state.
func (c *Setup) HandleAction(ctx context.Context, msg message.Message) {
	switch m := msg.(type) {
	case *SetupFinalizeMesssage:
		//TODO: validate setup. if it is invalid just provide an AutoSetup. Fuck the cheaters.
		c.wordSelections[m.UserId] = m.SelectedWords
		c.wordPlacements[m.UserId] = m.WordPlacements

		if c.AllPlayersDone() {
			for _, id := range c.g.Players {
				c.g.sendToClient(ctx, id, NewStartOrderDeciderMessage())
				opponentId := c.g.OtherPlayerId(id)
				c.g.Gameboards[opponentId] = NewGameboard(c.wordPlacements[id], c.wordSelections[id])
			}
			c.g.NextPhase(ctx)
		}

	default:
		log.Println("received invalid action during connecting phase -", m)
	}
}

func (c *Setup) AllPlayersDone() bool {
	for _, id := range c.g.Players {
		if _, ok := c.wordSelections[id]; !ok {
			return false
		}
	}
	return true
}

func (c *Setup) HandleDisconnect(ctx context.Context, msg message.Message) {
	switch m := msg.(type) {
	case *message.ClientQuitMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	case *message.ClientDisconnectMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	}
	c.g.Shutdown()
}

type TurnDecider struct {
	g             *Game
	playerResults map[uuidstring.ID]int
}

func NewTurnDecider(g *Game) *TurnDecider {
	playerResults := make(map[uuidstring.ID]int)
	for _, id := range g.Players {
		playerResults[id] = -1
	}
	return &TurnDecider{
		g:             g,
		playerResults: playerResults,
	}
}

func (c *TurnDecider) Enter(ctx context.Context) {
}

// HandleAction implements the GameState interface for TurnDecider state.
func (c *TurnDecider) HandleAction(ctx context.Context, msg message.Message) {
	// TODO: Add logic for handling actions in the TurnDecider state.
	switch m := msg.(type) {
	case *TurnDeciderUIEventMessage:
		uiEvent, err := message.UnmarshalWrappedType(m.UIEvent, uiEventTypeRegistry)
		if err != nil {
			log.Printf("failed to unmarshal ui event - %v\n", err)
			return
		}
		switch e := uiEvent.(type) {
		case *KeyPressedEvent:
			msg, err := NewTurnDeciderKeyPressedUpdateMessage(e.Key, m.UserId)
			if err != nil {
				log.Printf("failed to encode key pressed update message with err - %v\n", err)
				return
			}
			c.g.sendToClient(ctx, c.g.OtherPlayerId(m.UserId), msg)
		case *GuessedWordEvent:
			otherPlayerId := c.g.OtherPlayerId(m.UserId)
			otherPlayerResult := c.playerResults[otherPlayerId]
			playerResult := c.playerResults[m.UserId]
			if playerResult != -1 {
				return
			}

			letterGameStatuses, playerResult := c.g.Gameboards[m.UserId].ProcessTurnDeciderWordGuess(e.Word)
			c.playerResults[m.UserId] = playerResult
			turnDeciderResults := make(map[uuidstring.ID]TurnDeciderResult)

			var isFirstTurnDecided bool

			if otherPlayerResult == -1 {
				turnDeciderResults[m.UserId] = TurnDeciderWaitingForOther
				turnDeciderResults[otherPlayerId] = TurnDeciderOtherPlayerWent
			} else if playerResult < otherPlayerResult {
				c.g.FirstTurnWinner = otherPlayerId
				turnDeciderResults[m.UserId] = TurnDeciderLost
				turnDeciderResults[otherPlayerId] = TurnDeciderWin
				isFirstTurnDecided = true
			} else if playerResult > otherPlayerResult {
				c.g.FirstTurnWinner = m.UserId
				turnDeciderResults[m.UserId] = TurnDeciderWin
				turnDeciderResults[otherPlayerId] = TurnDeciderLost
				isFirstTurnDecided = true
			} else {
				turnDeciderResults[m.UserId] = TurnDeciderTie
				turnDeciderResults[otherPlayerId] = TurnDeciderTie
			}
			res, err := NewTurnDeciderTurnTakenUpdateMessage(letterGameStatuses, turnDeciderResults, m.UserId)
			if err != nil {
				log.Printf("failed to encode key pressed update message with err - %v\n", err)
				return
			}

			for _, pId := range c.g.Players {
				err := c.g.sendToClient(ctx, pId, res)
				if err != nil {
					log.Printf("failed to send to client %s with err - %v\n", pId, err)
					return
				}
			}
			if isFirstTurnDecided {
				c.g.NextPhase(ctx)
			} else if otherPlayerResult == -1 {
			} else {
				c.playerResults[m.UserId] = -1
				c.playerResults[otherPlayerId] = -1
			}
		case *UncoveredTileEvent:
			//TODO: process guessed word event, finding out how many tiles they uncovered, figuring out who won (if  both have guessed)
			// 		and then returning TurnDeciderUIUpdate message with who won the phase, or tied if tied
		}
	default:
		log.Println("received invalid action during connecting phase -", m)
	}
}

func (c *TurnDecider) HandleDisconnect(ctx context.Context, msg message.Message) {
	switch m := msg.(type) {
	case *message.ClientQuitMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	case *message.ClientDisconnectMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	}
	c.g.Shutdown()
}

type Gameplay struct {
	g                   *Game
	CurrentTurnPlayerId uuidstring.ID
	PlayerTimes         map[uuidstring.ID]uint
	PlayerTimers        map[uuidstring.ID]*time.Ticker
}

func NewGameplay(g *Game) *Gameplay {
	game := &Gameplay{
		g:                   g,
		CurrentTurnPlayerId: "",
		PlayerTimes: map[uuidstring.ID]uint{
			g.Players[0]: GameTimeLimit,
			g.Players[1]: GameTimeLimit,
		},
		PlayerTimers: map[uuidstring.ID]*time.Ticker{
			g.Players[0]: time.NewTicker(GameUpdateDuration),
			g.Players[1]: time.NewTicker(GameUpdateDuration),
		},
	}
	game.PlayerTimers[g.Players[0]].Stop()
	game.PlayerTimers[g.Players[1]].Stop()

	return game
}

func (c *Gameplay) Enter(ctx context.Context) {
	c.CurrentTurnPlayerId = c.g.FirstTurnWinner
	log.Printf("starting gameplay, first turn is player[%s]\n", c.CurrentTurnPlayerId)
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("Game[%s] timer loop exiting due to context being cancelled - player[%s] and player[%s]", c.g.GameID, c.g.Players[0], c.g.Players[1])
				return
			case <-c.PlayerTimers[c.g.Players[0]].C:
				c.PlayerTimes[c.g.Players[0]]--
				var turnResult TurnResult
				if c.PlayerTimes[c.g.Players[0]] == 0 {
					turnResult = Loss
				} else {
					turnResult = NoChange
				}
				msg, err := NewGameplayTimeUpdateMessage(c.PlayerTimes[c.g.Players[0]], c.g.Players[0], turnResult)
				if err != nil {
					fmt.Printf("failed to create TimeUpdateMessage - %v", err)
					continue
				}
				for _, playerId := range c.g.Players {
					c.g.sendToClient(ctx, playerId, msg)
				}
			case <-c.PlayerTimers[c.g.Players[1]].C:
				c.PlayerTimes[c.g.Players[1]]--
				var turnResult TurnResult
				if c.PlayerTimes[c.g.Players[1]] == 0 {
					turnResult = Loss
				} else {
					turnResult = NoChange
				}
				msg, err := NewGameplayTimeUpdateMessage(c.PlayerTimes[c.g.Players[1]], c.g.Players[1], turnResult)
				if err != nil {
					fmt.Printf("failed to create TimeUpdateMessage - %v", err)
					continue
				}
				for _, playerId := range c.g.Players {
					c.g.sendToClient(ctx, playerId, msg)
				}
			}
		}
	}()
	c.StartPlayerTurn(c.g.FirstTurnWinner, c.g.OtherPlayerId(c.g.FirstTurnWinner))
}

// HandleAction implements the GameState interface for Gameplay state.
func (c *Gameplay) HandleAction(ctx context.Context, msg message.Message) {
	switch m := msg.(type) {
	case *GameplayUIEventMessage:
		uiEvent, err := message.UnmarshalWrappedType(m.UIEvent, uiEventTypeRegistry)
		if err != nil {
			log.Printf("failed to unmarshal ui event - %v\n", err)
			return
		}
		switch e := uiEvent.(type) {
		case *KeyPressedEvent:
			msg, err := NewGameplayKeyPressedUpdateMessage(e.Key, m.UserId)
			if err != nil {
				log.Printf("failed to encode key pressed update message with err - %v\n", err)
				return
			}
			c.g.sendToClient(ctx, c.g.OtherPlayerId(m.UserId), msg)
		case *GuessedWordEvent:
			if m.UserId != c.CurrentTurnPlayerId {
				log.Printf("Wrong user[%s] is trying to take their turn when it is User[%s]'s turn\n", m.UserId, c.CurrentTurnPlayerId)
			}
			letterGameStatuses, turnResult := c.g.Gameboards[m.UserId].ProcessGameplayWordGuess(e.Word)

			msg, err := NewGameplayTurnTakenUpdateMessage(letterGameStatuses, turnResult, m.UserId)
			if err != nil {
				log.Printf("failed to encode key pressed update message with err - %v\n", err)
				return
			}
			for _, playerId := range c.g.Players {
				c.g.sendToClient(ctx, playerId, msg)
			}
			if turnResult == Win {
				c.g.ProcessGameResult(m.UserId)
				c.g.NextPhase(ctx)
			} else {
				c.StartPlayerTurn(c.g.OtherPlayerId(m.UserId), m.UserId)
			}
		case *UncoveredTileEvent:
			if m.UserId != c.CurrentTurnPlayerId {
				log.Printf("Wrong user[%s] is trying to take their turn when it is User[%s]'s turn\n", m.UserId, c.CurrentTurnPlayerId)
			}
			letterGameStatuses, turnResult := c.g.Gameboards[m.UserId].ProcessGameplayTileUncover(e.Row, e.Col)

			msg, err := NewGameplayTurnTakenUpdateMessage(letterGameStatuses, turnResult, m.UserId)
			if err != nil {
				log.Printf("failed to encode key pressed update message with err - %v\n", err)
				return
			}
			for _, playerId := range c.g.Players {
				c.g.sendToClient(ctx, playerId, msg)
			}
			if turnResult == Win {
				c.g.ProcessGameResult(m.UserId)
				c.g.NextPhase(ctx)
			} else {
				c.StartPlayerTurn(c.g.OtherPlayerId(m.UserId), m.UserId)
			}
		}
	}
}

func (c *Gameplay) HandleDisconnect(ctx context.Context, msg message.Message) {
	switch m := msg.(type) {
	case *message.ClientQuitMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	case *message.ClientDisconnectMessage:
		otherPlayerId := c.g.OtherPlayerId(m.UserId)
		err := c.g.sendToClient(ctx, otherPlayerId, NewPlayerQuitMessage(m.UserId))
		if err != nil {
			log.Printf("failed to send player quit message to other client with error - %v", err)
			return
		}
	}
	c.g.Shutdown()
}

func (c *Gameplay) StartPlayerTurn(playerTurnId uuidstring.ID, playerNonTurnId uuidstring.ID) {
	c.CurrentTurnPlayerId = playerTurnId
	c.PlayerTimers[playerNonTurnId].Stop()
	c.PlayerTimers[playerTurnId].Reset(GameUpdateDuration)
}

type Postgame struct {
	g                 *Game
	playerDisconnects map[uuidstring.ID]bool
}

func NewPostgame(g *Game) *Postgame {
	p := &Postgame{
		g:                 g,
		playerDisconnects: map[uuidstring.ID]bool{},
	}
	for _, playerId := range g.Players {
		p.playerDisconnects[playerId] = false
	}
	return p
}

func (c *Postgame) Enter(ctx context.Context) {}

// HandleAction implements the GameState interface for Postgame state.
func (c *Postgame) HandleAction(ctx context.Context, msg message.Message) {
	// TODO: Add logic for handling actions in the Postgame state.
}

func (c *Postgame) HandleDisconnect(ctx context.Context, msg message.Message) {
	//TODO: log each player disconnecting, once both have disconnected trigger the game to close, maybe delete room?
	var disconnectedPlayerId uuidstring.ID
	switch m := msg.(type) {
	case *message.ClientDisconnectMessage:
		disconnectedPlayerId = m.UserId
	case *message.ClientQuitMessage:
		disconnectedPlayerId = m.UserId
	}
	c.playerDisconnects[disconnectedPlayerId] = true
	didAllDisconnect := true
	for playerId, didDisconnect := range c.playerDisconnects {
		if !didDisconnect {
			didAllDisconnect = false
			c.g.sendToClient(ctx, playerId, message.NewClientQuitMessage(disconnectedPlayerId))
		}
	}
	if didAllDisconnect {
		c.g.Shutdown()
	}
}

func allPlayersConnected(playersConnected *map[uuidstring.ID]bool) bool {
	for _, isConnected := range *playersConnected {
		if !isConnected {
			return false
		}
	}
	return true
}
