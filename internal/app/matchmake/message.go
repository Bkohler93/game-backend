package matchmake

import (
	"context"

	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

type PlayerLeftRoom struct {
	TypeDiscriminator string                `json:"$type"`
	UserLeftId        stringuuid.StringUUID `json:"user_left_id"`
}

func EmptyPlayerLeftRoomMessage() PlayerLeftRoom {
	return PlayerLeftRoom{
		TypeDiscriminator: message.PrintTypeDiscriminator(PlayerLeftRoom{}),
	}
}

type PlayerJoinedRoom struct {
	TypeDiscriminator string                `json:"$type"`
	UserJoinedId      stringuuid.StringUUID `json:"user_joined_id"`
}

func NewPlayerJoinedRoomMessage(userJoinedId stringuuid.StringUUID) PlayerJoinedRoom {
	return PlayerJoinedRoom{
		TypeDiscriminator: message.PrintTypeDiscriminator(PlayerJoinedRoom{}),
		UserJoinedId:      userJoinedId,
	}
}

type RoomChanged struct {
	TypeDiscriminator string                `json:"$type"`
	NewRoomId         stringuuid.StringUUID `json:"new_room_id"`
	PlayerCount       int                   `json:"player_count"`
	AvgSkill          int                   `json:"avg_skill"`
}

func NewRoomChangedMessage(emptyRoomId stringuuid.StringUUID, playerCount, avgSkill int) RoomChanged {
	return RoomChanged{
		TypeDiscriminator: message.PrintTypeDiscriminator(RoomChanged{}),
		NewRoomId:         emptyRoomId,
		PlayerCount:       playerCount,
		AvgSkill:          avgSkill,
	}
}

type MatchmakingClientMessageProducer interface {
	Publish(ctx context.Context, userId stringuuid.StringUUID, data any) error
}

type MatchmakingClientMessageRedisProducer struct {
	rdb *redis.Client
}

func NewMatchmakingClientMessageRedisProducer(rdb *redis.Client) *MatchmakingClientMessageRedisProducer {
	m := &MatchmakingClientMessageRedisProducer{rdb: rdb}
	return m
}

func (m MatchmakingClientMessageRedisProducer) Publish(ctx context.Context, userId stringuuid.StringUUID, data any) error {
	stream := rediskeys.MatchmakingClientMessageStream(userId)
	return m.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: data,
	}).Err()
}
