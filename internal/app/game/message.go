package game

import (
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

	GameStart            GameClientMessageType = "GameStart" //initial game start
	GameFirstGuessResult GameClientMessageType = "GameFirstGuessResult"
	GameOrderResult      GameClientMessageType = "GameOrderResult" //includes data that tells whether to re-do first guesses or whose turn it is to go first

	GameGuessResult GameClientMessageType = "GameGuessResult"
)

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

type GameServerMessage interface {
	message.Message
}

type GameServerMessageType string

const (
	ConfirmMatch GameServerMessageType = "ConfirmMatch"

	SetupPlacementAttempt GameServerMessageType = "SetupPlacementAttempt"
	SetupUndo             GameServerMessageType = "SetupUndo"
	SetupFinalize         GameServerMessageType = "SetupFinalize"

	GameFirstGuess GameServerMessageType = "GameFirstGuess"

	GameGuess GameServerMessageType = "GameGuess"
)

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
