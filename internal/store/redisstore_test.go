package store

import (
	"testing"

	"github.com/bkohler93/game-backend/internal/models"
	"github.com/redis/go-redis/v9"
)

var client = newRedisClient()
var store = NewRediStore(client)

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		DB:       0, // use default DB
		Password: "",
		Protocol: 2,
	})
}

func TestCreateRoomIndex(t *testing.T) {
	ctx := t.Context()
	err := store.CreateRoomIndex(ctx)
	if err != nil && err.Error() != "Index already exists" {
		t.Errorf("failed to create index - %v", err)
	}

	info, err := client.FTInfo(ctx, RoomIndex).Result()
	if err != nil {
		t.Errorf("should not result in an error when retrieving index info - %v", err)
	}
	if info.IndexName != RoomIndex {
		t.Errorf("index info name should be %s, got %s", RoomIndex, info.IndexName)
	}
}

func TestStoreRoom(t *testing.T) {
	ctx := t.Context()
	r := models.RandomRoom()
	err := store.StoreRoom(ctx, r)
	if err != nil {
		t.Errorf("error storing room - %v", err)
	}
}
