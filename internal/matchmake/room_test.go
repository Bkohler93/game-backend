package matchmake

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/models"
	"github.com/bkohler93/game-backend/internal/store"
	"github.com/bkohler93/game-backend/internal/utils/redisutils"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

func TestRedisJsonRoom(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		DB:       0, // use default DB
		Password: "",
		Protocol: 2,
	})
	s := store.NewRediStore(rdb)
	ctx := t.Context()

	r1 := models.Room{
		RoomId:       stringuuid.NewStringUUID(),
		PlayerCount:  1,
		AverageSkill: 100,
		Region:       "na",
		PlayerIds: []stringuuid.StringUUID{
			stringuuid.NewStringUUID(),
		},
		CreatedAt: time.Now().Add(time.Second * 30 * -1).Unix(),
	}

	r2 := models.Room{
		RoomId:       stringuuid.NewStringUUID(),
		PlayerCount:  1,
		AverageSkill: 105,
		Region:       "na",
		PlayerIds: []stringuuid.StringUUID{
			stringuuid.NewStringUUID(),
		},
		CreatedAt: time.Now().Add(time.Second * 30 * -2).Unix(),
	}
	err := s.CreateRoomIndex(ctx)
	if err != nil {
		t.Errorf("createRoomIndex should not result in an error. Got - %v", err)
	}

	err = s.StoreRoom(ctx, r1)
	if err != nil {
		t.Errorf("StoreRoom(r1) should not result in an error. Got - %v", err)
	}
	err = s.StoreRoom(ctx, r2)
	if err != nil {
		t.Errorf("StoreRoom(r2) should not result in an error. Got - %v", err)
	}
	_, err = rdb.JSONSet(ctx, "room:1", "$", r1).Result()
	if err != nil {
		t.Errorf("failed to set json object in redis - %v", err)
	}

	_, err = rdb.JSONSet(context.Background(), "room:2", "$", r2).Result()
	if err != nil {
		t.Errorf("failed to set json object in redis - %v", err)
	}

	// d, err := rdb.JSONGet(ctx, "room:1", "created_at").Result()
	// if err != nil {
	// 	t.Errorf("failed to retrieve created_at for room1 - %v", err)
	// }

	// secs, err := strconv.ParseInt(d, 10, 64)
	// if err != nil {
	// 	t.Errorf("failed to parse unix time into seconds - %v", err)
	// }
	// tim := time.Unix(secs, 0)

	now := time.Now()
	mySkill := 110
	region := "na"

	minSkill, maxSkill := calculateMinMaxSkill(mySkill, now)
	maxPlayerCount := 3 - 1

	findMatchResult, err := rdb.FTSearch(
		ctx,
		"idx:rooms",
		fmt.Sprintf("@average_skill:[%d %d] @region:%s @player_count:[0,%d]", minSkill, maxSkill, region, maxPlayerCount),
	).Result()
	if err != nil {
		t.Errorf("failed to retrieve query - %v", err)
	}

	rooms, err := redisutils.SortedSliceFromRedisDocs(findMatchResult.Docs, func(a, b models.Room) int {
		if time.Unix(a.CreatedAt, 0).Before(time.Unix(b.CreatedAt, 0)) {
			return -1
		}
		return 1
	})
	if err != nil {
		t.Errorf("failed to unmarshal sorted slice of rooms from match result docs - %v", err)
	}

	if rooms[0].AverageSkill < rooms[1].AverageSkill {
		t.Errorf("incorrect order of rooms... rooms[0].createdAt={%s}, rooms[1].createdAt={%s}", time.Unix(rooms[0].CreatedAt, 0).String(), time.Unix(rooms[1].CreatedAt, 0).String())
	}
}

func TestCalculateThreshold(t *testing.T) {
	t.Run("test 31 second old time", func(t *testing.T) {
		tm := time.Now().Add(time.Second * -31)

		expected := 30
		actual := calculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})

	t.Run("test 0 second old time", func(t *testing.T) {
		tm := time.Now()

		expected := BaseSkillThreshold
		actual := calculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})

	t.Run("test 61 second old time", func(t *testing.T) {
		tm := time.Now().Add(time.Second * -61)

		expected := 50
		actual := calculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})
}
