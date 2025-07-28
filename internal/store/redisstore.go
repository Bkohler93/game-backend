package store

import (
	"context"
	"fmt"

	"github.com/bkohler93/game-backend/internal/models"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

const (
	RoomIndex = "idx:rooms"
)

type RedisStore struct {
	*redis.Client
}

func NewRediStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{
		rdb,
	}
}

func (rs *RedisStore) CreateRoomIndex(ctx context.Context) error {
	return rs.FTCreate(
		ctx,
		RoomIndex,
		// Options:
		&redis.FTCreateOptions{
			OnJSON: true,
			Prefix: []interface{}{"room:"},
		},
		// Index schema fields:
		&redis.FieldSchema{
			FieldName: "$.room_id",
			As:        "room_id",
			FieldType: redis.SearchFieldTypeText,
		},
		&redis.FieldSchema{
			FieldName: "$.player_count",
			As:        "player_count",
			FieldType: redis.SearchFieldTypeNumeric,
		},
		&redis.FieldSchema{
			FieldName: "$.average_skill",
			As:        "average_skill",
			FieldType: redis.SearchFieldTypeNumeric,
		},
		&redis.FieldSchema{
			FieldName: "$.region",
			As:        "region",
			FieldType: redis.SearchFieldTypeText,
		},
		&redis.FieldSchema{
			FieldName: "$.created_at",
			As:        "created_at",
			FieldType: redis.SearchFieldTypeNumeric,
		},
	).Err()
}

func (rs *RedisStore) StoreRoom(ctx context.Context, r models.Room) error {
	id := stringuuid.NewStringUUID()
	key := fmt.Sprintf("room:%s", id.String())
	return rs.JSONSet(ctx, key, "$", r).Err()
}

func (rs *RedisStore) StoreKeyValue(ctx context.Context, key string, value any) error {
	_, err := rs.HSet(ctx, key, value).Result()
	return err
}
