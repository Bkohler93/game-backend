package message

import (
	"encoding/json"
	"reflect"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

var printTypeDiscriminator = func(i any) string {
	return reflect.TypeOf(i).String()
}

type MatchmakingClientMessageType string

const (
	ClientMessageTypeMatchMade        MatchmakingClientMessageType = "MatchMade"
	ClientMessageTypePlayerLeftRoom   MatchmakingClientMessageType = "PlayerLeftRoom"
	ClientMessageTypePlayerJoinedRoom MatchmakingClientMessageType = "PlayerJoinedRoom"
	ClientMessageTypeRoomChanged      MatchmakingClientMessageType = "RoomChanged"
)

type BaseMatchmakingClientMessage struct {
	Type    MatchmakingClientMessageType `json:"type"`
	Payload json.RawMessage              `json:"payload"`
}

type MatchMade struct {
	TypeDiscriminator string                `json:"$type"`
	UserOneId         stringuuid.StringUUID `redis:"user_one_id" json:"user_one_id"`
	UserOneName       string                `redis:"user_one_name" json:"user_one_name"`
	UserTwoId         stringuuid.StringUUID `redis:"user_two_id" json:"user_two_id"`
	UserTwoName       string                `redis:"user_two_name" json:"user_two_name"`
}

func EmptyMatchMadeMessage() MatchMade {
	return MatchMade{
		TypeDiscriminator: printTypeDiscriminator(MatchMade{}),
	}
}

type PlayerLeftRoom struct {
	TypeDiscriminator string                `json:"$type"`
	UserLeftId        stringuuid.StringUUID `json:"user_left_id"`
}

func EmptyPlayerLeftRoomMessage() PlayerLeftRoom {
	return PlayerLeftRoom{
		TypeDiscriminator: printTypeDiscriminator(PlayerLeftRoom{}),
	}
}

type PlayerJoinedRoom struct {
	TypeDiscriminator string `json:"$type"`
	UserJoinedId      string `json:"user_joined_id"`
}

func EmptyPlayerJoinedRoomMessage() PlayerJoinedRoom {
	return PlayerJoinedRoom{
		TypeDiscriminator: printTypeDiscriminator(PlayerJoinedRoom{}),
	}
}

type RoomChanged struct {
	TypeDiscriminator string                `json:"$type"`
	EmptyRoomId       stringuuid.StringUUID `json:"new_room_id"`
	PlayerCount       int                   `json:"player_count"`
	AvgSkill          int                   `json:"avg_skill"`
}

func EmptyRoomChangedMessage() RoomChanged {
	return RoomChanged{
		TypeDiscriminator: printTypeDiscriminator(RoomChanged{}),
	}
}
