package matchmake

import (
	"encoding/json"
	"fmt"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

type MatchmakingClientMessage interface {
	message.Discriminable
}

func UnmarshalMatchmakingClientMessage(data []byte) (MatchmakingClientMessage, error) {
	var temp struct {
		TypeDiscriminator string `json:"$type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return nil, err
	}

	if constructor, ok := matchmakingMessageRegistry[temp.TypeDiscriminator]; ok {
		concreteMessage := constructor()
		if err := json.Unmarshal(data, &concreteMessage); err != nil {
			return nil, err
		}
		return concreteMessage, nil
	}

	return nil, fmt.Errorf("unknown matchmaking message type: %s", temp.TypeDiscriminator)
}

var matchmakingMessageRegistry = map[string]func() MatchmakingClientMessage{
	"PlayerLeftRoomMessage":   func() MatchmakingClientMessage { return &PlayerLeftRoomMessage{} },
	"PlayerJoinedRoomMessage": func() MatchmakingClientMessage { return &PlayerJoinedRoomMessage{} },
	"RoomChangedMessage":      func() MatchmakingClientMessage { return &RoomChangedMessage{} },
}

type MatchmakingClientMessageType string

const (
	PlayerLeftRoom   MatchmakingClientMessageType = "PlayerLeftRoom"
	PlayerJoinedRoom MatchmakingClientMessageType = "PlayerJoinedRoom"
	RoomChanged      MatchmakingClientMessageType = "RoomChanged"
)

type PlayerLeftRoomMessage struct {
	TypeDiscriminator string                `json:"$type"`
	UserLeftId        stringuuid.StringUUID `json:"user_left_id"`
}

func (p PlayerLeftRoomMessage) GetDiscriminator() string {
	return p.TypeDiscriminator
}

func NewPlayerLeftRoomMessage(userLeftId stringuuid.StringUUID) PlayerLeftRoomMessage {
	return PlayerLeftRoomMessage{
		TypeDiscriminator: message.PrintTypeDiscriminator(PlayerLeftRoomMessage{}),
		UserLeftId:        userLeftId,
	}
}

type PlayerJoinedRoomMessage struct {
	TypeDiscriminator string                `json:"$type"`
	UserJoinedId      stringuuid.StringUUID `json:"user_joined_id"`
}

func (p PlayerJoinedRoomMessage) GetDiscriminator() string {
	return p.TypeDiscriminator
}

func NewPlayerJoinedRoomMessage(userJoinedId stringuuid.StringUUID) PlayerJoinedRoomMessage {
	return PlayerJoinedRoomMessage{
		TypeDiscriminator: "PlayerJoinedRoomMessage",
		UserJoinedId:      userJoinedId,
	}
}

type RoomChangedMessage struct {
	TypeDiscriminator string                `json:"$type"`
	NewRoomId         stringuuid.StringUUID `json:"new_room_id"`
	PlayerCount       int                   `json:"player_count"`
	AvgSkill          int                   `json:"avg_skill"`
}

func (r RoomChangedMessage) GetDiscriminator() string {
	return r.TypeDiscriminator
}

func NewRoomChangedMessage(emptyRoomId stringuuid.StringUUID, playerCount, avgSkill int) RoomChangedMessage {
	return RoomChangedMessage{
		TypeDiscriminator: message.PrintTypeDiscriminator(RoomChangedMessage{}),
		NewRoomId:         emptyRoomId,
		PlayerCount:       playerCount,
		AvgSkill:          avgSkill,
	}
}
