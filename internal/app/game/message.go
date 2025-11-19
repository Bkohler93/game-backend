package game

import (
	"encoding/json"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type GameClientMessage interface {
	message.Message
}

type GameClientMessageType string

const (
	StartSetup            GameClientMessageType = "StartSetup" //contains 9 words for players to choose from
	SetupPlacementSuccess GameClientMessageType = "SetupPlacementSuccess"
	SetupPlacementFail    GameClientMessageType = "SetupPlacementFail"
	SetupUndoResult       GameClientMessageType = "SetupUndoResult"

	StartOrderDecider   GameClientMessageType = "StartOrderDecider"
	TurnDeciderUIUpdate GameClientMessageType = "TurnDeciderUIUpdate"
	GameplayUIUpdate    GameClientMessageType = "GameplayUIUpdate"

	PlayerQuit GameClientMessageType = "PlayerQuit"
	GameOver   GameClientMessageType = "GameOver"
	Emote      GameClientMessageType = "Emote"
)

type PlayerQuitMessage struct {
	TypeDiscriminator string        `json:"$type"`
	UserId            uuidstring.ID `json:"user_id"`
}

func (m *PlayerQuitMessage) GetDiscriminator() string {
	return string(PlayerQuit)
}

func NewPlayerQuitMessage(userId uuidstring.ID) *PlayerQuitMessage {
	return &PlayerQuitMessage{
		UserId:            userId,
		TypeDiscriminator: string(PlayerQuit),
	}
}

type StartOrderDeciderMessage struct {
	TypeDiscriminator string `json:"$type"`
}

func NewStartOrderDeciderMessage() *StartOrderDeciderMessage {
	return &StartOrderDeciderMessage{TypeDiscriminator: string(StartOrderDecider)}
}

func (m *StartOrderDeciderMessage) GetDiscriminator() string {
	return string(StartOrderDecider)
}

type StartSetupMessage struct {
	TypeDiscriminator string   `json:"$type"`
	ThreeLetterWords  []string `json:"three_letter_words"`
	FourLetterWords   []string `json:"four_letter_words"`
	FiveLetterWords   []string `json:"five_letter_words"`
}

func NewStartSetupMessage(threeLetterWords, fourLetterWords, fiveLetterWords []string) *StartSetupMessage {
	return &StartSetupMessage{
		TypeDiscriminator: string(StartSetup),
		ThreeLetterWords:  threeLetterWords,
		FourLetterWords:   fourLetterWords,
		FiveLetterWords:   fiveLetterWords,
	}
}

func (m *StartSetupMessage) GetDiscriminator() string {
	return string(StartSetup)
}

type UIUpdate interface {
	message.Message
}

type UIUpdateType string

const (
	TurnTakenUpdate  UIUpdateType = "TurnTakenUpdate"
	KeyPressedUpdate UIUpdateType = "KeyPressedUpdate"
	TimeUpdate       UIUpdateType = "TimeUpdate"
)

type TimeUpdateMessage struct {
	TypeDiscriminator string `json:"$type"`
	TimeLeft          uint   `json:"time_left"`
}

func NewGameplayTimeUpdateMessage(timeLeft uint, userID uuidstring.ID, turnResult TurnResult) (*GameplayUIUpdateMessage, error) {
	timeUpdateMessage := &TimeUpdateMessage{
		TypeDiscriminator: string(TimeUpdate),
		TimeLeft:          timeLeft,
	}

	bytes, err := json.Marshal(timeUpdateMessage)
	if err != nil {
		return nil, err
	}

	return &GameplayUIUpdateMessage{
		TypeDiscriminator: string(GameplayUIUpdate),
		UserId:            userID,
		UIUpdate:          bytes,
		TurnResult:        turnResult,
	}, nil
}

type KeyPressedUpdateMessage struct {
	TypeDiscriminator string `json:"$type"`
	Key               string `json:"key"`
}

func NewTurnDeciderKeyPressedUpdateMessage(k string, userID uuidstring.ID) (*TurnDeciderUIUpdateMessage, error) {
	keyPressedMessage := &KeyPressedUpdateMessage{
		TypeDiscriminator: string(KeyPressedUpdate),
		Key:               k,
	}

	bytes, err := json.Marshal(keyPressedMessage)
	if err != nil {
		return nil, err
	}

	return &TurnDeciderUIUpdateMessage{
		TypeDiscriminator:  string(TurnDeciderUIUpdate),
		UserId:             userID,
		UIUpdate:           bytes,
		TurnDeciderResults: nil,
	}, nil
}

func NewGameplayKeyPressedUpdateMessage(k string, userID uuidstring.ID) (*GameplayUIUpdateMessage, error) {
	keyPressedMessage := &KeyPressedUpdateMessage{
		TypeDiscriminator: string(KeyPressedUpdate),
		Key:               k,
	}

	bytes, err := json.Marshal(keyPressedMessage)
	if err != nil {
		return nil, err
	}

	return &GameplayUIUpdateMessage{
		TypeDiscriminator: string(GameplayUIUpdate),
		UserId:            userID,
		UIUpdate:          bytes,
		TurnResult:        NoChange,
	}, nil
}

type TurnTakenUpdateMessage struct {
	TypeDiscriminator  string          `json:"$type"`
	LetterGameStatuses GameStateUpdate `json:"letter_game_statuses"`
}

func NewTurnDeciderTurnTakenUpdateMessage(letterGameStatuses GameStateUpdate, turnDeciderResults map[uuidstring.ID]TurnDeciderResult, userID uuidstring.ID) (*TurnDeciderUIUpdateMessage, error) {
	turnTakenMessage := &TurnTakenUpdateMessage{
		TypeDiscriminator:  string(TurnTakenUpdate),
		LetterGameStatuses: letterGameStatuses,
	}

	bytes, err := json.Marshal(turnTakenMessage)
	if err != nil {
		return nil, err
	}
	return &TurnDeciderUIUpdateMessage{
		TypeDiscriminator:  string(TurnDeciderUIUpdate),
		UserId:             userID,
		UIUpdate:           bytes,
		TurnDeciderResults: turnDeciderResults,
	}, nil
}

func NewGameplayTurnTakenUpdateMessage(letterGameStatuses GameStateUpdate, turnResult TurnResult, userId uuidstring.ID) (*GameplayUIUpdateMessage, error) {
	turnTakenMessage := &TurnTakenUpdateMessage{
		TypeDiscriminator:  string(TurnTakenUpdate),
		LetterGameStatuses: letterGameStatuses,
	}

	bytes, err := json.Marshal(turnTakenMessage)
	if err != nil {
		return nil, err
	}
	return &GameplayUIUpdateMessage{
		TypeDiscriminator: string(GameplayUIUpdate),
		UserId:            userId,
		UIUpdate:          bytes,
		TurnResult:        turnResult,
	}, nil
}

func (m *TurnTakenUpdateMessage) GetDiscriminator() string {
	return string(TurnTakenUpdate)
}

type TurnDeciderUIUpdateMessage struct {
	TypeDiscriminator  string                              `json:"$type"`
	UserId             uuidstring.ID                       `json:"user_id"`
	UIUpdate           json.RawMessage                     `json:"ui_update"`
	TurnDeciderResults map[uuidstring.ID]TurnDeciderResult `json:"turn_decider_results"`
}

func (m *TurnDeciderUIUpdateMessage) GetDiscriminator() string {
	return string(TurnDeciderUIUpdate)
}

type TurnDeciderResult int

const (
	TurnDeciderWin TurnDeciderResult = iota
	TurnDeciderLost
	TurnDeciderTie
	TurnDeciderWaitingForOther
	TurnDeciderOtherPlayerWent
)

type GameplayUIUpdateMessage struct {
	TypeDiscriminator string          `json:"$type"`
	UserId            uuidstring.ID   `json:"user_id"`
	UIUpdate          json.RawMessage `json:"ui_update"`
	TurnResult        TurnResult      `json:"turn_result"`
}

func (m *GameplayUIUpdateMessage) GetDiscriminator() string {
	return string(GameplayUIUpdate)
}

type TurnResult int

const (
	GoAgain TurnResult = iota
	EndTurn
	Win
	Loss
	NoChange
)

type EmoteMessage struct {
	TypeDiscriminator string `json:"$type"`
}

func NewEmoteMessage() *EmoteMessage {
	return &EmoteMessage{
		TypeDiscriminator: string(Emote),
	}
}

func (m *EmoteMessage) GetDiscriminator() string {
	return string(Emote)
}

type GameServerMessage interface {
	message.Message
}

type GameServerMessageType string

const (
	ConfirmMatch GameServerMessageType = "ConfirmMatch"

	// SetupPlacementAttempt GameServerMessageType = "SetupPlacementAttempt"
	// SetupUndo             GameServerMessageType = "SetupUndo"
	SetupFinalize GameServerMessageType = "SetupFinalize"

	TurnDeciderUIEvent GameServerMessageType = "TurnDeciderUIEventMessage"
	GameplayUIEvent    GameServerMessageType = "GameplayUIEventMessage"

	ClientEmote GameServerMessageType = "ClientEmote"
)

var gameServerMessageTypes map[string]struct{} = map[string]struct{}{
	string(ConfirmMatch):             {},
	string(SetupFinalize):            {},
	string(ClientEmote):              {},
	string(GameplayUIEvent):          {},
	string(TurnDeciderUIEvent):       {},
	string(message.ClientDisconnect): {},
	string(message.ClientQuit):       {},
}

func IsGameServerMessageType(v string) bool {
	if _, ok := gameServerMessageTypes[v]; ok {
		return true
	}
	return false
}

var serverMessageTypeRegistry = map[string]func() message.Message{
	string(ConfirmMatch):             func() message.Message { return &ConfirmMatchMessage{} },
	string(SetupFinalize):            func() message.Message { return &SetupFinalizeMesssage{} },
	string(GameplayUIEvent):          func() message.Message { return &GameplayUIEventMessage{} },
	string(TurnDeciderUIEvent):       func() message.Message { return &TurnDeciderUIEventMessage{} },
	string(message.ClientDisconnect): func() message.Message { return &message.ClientDisconnectMessage{} },
	string(message.ClientQuit):       func() message.Message { return &message.ClientQuitMessage{} },
	// string(SetupPlacementAttempt): func() GameServerMessage { return &SetupPlacementAttemptMessage{} },
}

type UIEvent interface {
	message.Message
}

type UIEventType string

const (
	KeyPressed    UIEventType = "KeyPressed"
	GuessedWord   UIEventType = "GuessedWord"
	UncoveredTile UIEventType = "UncoveredTile"
)

var uiEventTypeRegistry = map[string]func() UIEvent{
	string(KeyPressed):    func() UIEvent { return &KeyPressedEvent{} },
	string(GuessedWord):   func() UIEvent { return &GuessedWordEvent{} },
	string(UncoveredTile): func() UIEvent { return &UncoveredTileEvent{} },
}

type UncoveredTileEvent struct {
	TypeDiscriminator string `json:"$type"`
	Row               int    `json:"row"`
	Col               int    `json:"col"`
}

func (u *UncoveredTileEvent) GetDiscriminator() string {
	return string(UncoveredTile)
}

type GuessedWordEvent struct {
	TypeDiscriminator string `json:"$type"`
	Word              string `json:"word"`
}

func NewGuessedWordEvent() *GuessedWordEvent {
	return &GuessedWordEvent{
		TypeDiscriminator: string(GuessedWord),
	}
}

func (g *GuessedWordEvent) GetDiscriminator() string {
	return string(GuessedWord)
}

type KeyPressedEvent struct {
	TypeDiscriminator string `json:"$type"`
	Key               string `json:"key"`
}

func NewKeyPressedEvent() *KeyPressedEvent {
	return &KeyPressedEvent{
		TypeDiscriminator: string(KeyPressed),
	}
}

func (k *KeyPressedEvent) GetDiscriminator() string {
	return string(KeyPressed)
}

type TurnDeciderUIEventMessage struct {
	TypeDiscriminator string          `json:"$type"`
	UserId            uuidstring.ID   `json:"user_id"`
	UIEvent           json.RawMessage `json:"ui_event"`
}

// GetDiscriminator implements GameServerMessage.
func (t *TurnDeciderUIEventMessage) GetDiscriminator() string {
	return string(TurnDeciderUIEvent)
}

type GameplayUIEventMessage struct {
	TypeDiscriminator string          `json:"$type"`
	UserId            uuidstring.ID   `json:"user_id"`
	UIEvent           json.RawMessage `json:"ui_event"`
}

func NewGameUIEventMessage() *GameplayUIEventMessage {
	return &GameplayUIEventMessage{
		TypeDiscriminator: string(GameplayUIEvent),
	}
}

func (m *GameplayUIEventMessage) GetDiscriminator() string {
	return string(GameplayUIEvent)
}

type SetupFinalizeMesssage struct {
	TypeDiscriminator string          `json:"$type"`
	UserId            uuidstring.ID   `json:"user_id"`
	SelectedWords     *WordSelections `json:"selected_words"`
	WordPlacements    *WordPlacements `json:"word_placements"`
}

func (m *SetupFinalizeMesssage) GetDiscriminator() string {
	return string(SetupFinalize)
}

// type SetupPlacementAttemptMessage struct {
// 	TypeDiscriminator  string             `json:"$type"`
// 	StartingCol        int                `json:"starting_col"`
// 	StartingRow        int                `json:"starting_row"`
// 	PlacementDirection PlacementDirection `json:"placement_direction"`
// 	Word               string             `json:"word"`
// }

// func (s *SetupPlacementAttemptMessage) GetDiscriminator() string {
// 	return s.TypeDiscriminator
// }

// func NewSetupPlacementAttemptMessage(startingCol, startingRow int, placementDirection PlacementDirection, word string) *SetupPlacementAttemptMessage {
// 	return &SetupPlacementAttemptMessage{
// 		TypeDiscriminator:  string(SetupPlacementAttempt),
// 		StartingCol:        startingCol,
// 		StartingRow:        startingRow,
// 		PlacementDirection: placementDirection,
// 		Word:               word,
// 	}
// }

type ConfirmMatchMessage struct {
	TypeDiscriminator string        `json:"$type"`
	UserId            uuidstring.ID `json:"user_id"`
	RoomId            uuidstring.ID `json:"room_id"`
}

func NewConfirmMatchMessage(userId, roomId uuidstring.ID) *ConfirmMatchMessage {
	return &ConfirmMatchMessage{
		TypeDiscriminator: string(ConfirmMatch),
		UserId:            userId,
		RoomId:            roomId,
	}
}

func (m *ConfirmMatchMessage) GetDiscriminator() string {
	return m.TypeDiscriminator
}
