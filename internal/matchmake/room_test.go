package matchmake

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

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
	ctx := context.Background()

	r := Room{
		RoomId:       stringuuid.NewStringUUID(),
		PlayerCount:  1,
		AverageSkill: 100,
		Region:       "na",
		PlayerIds: []stringuuid.StringUUID{
			stringuuid.NewStringUUID(),
		},
		CreatedAt: time.Now().Add(time.Second * 30 * -1).Unix(),
	}

	_, err := rdb.FTCreate(
		ctx,
		"idx:rooms",
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
	).Result()

	if err != nil {
		t.Errorf("failed to set index - %v", err)
	}

	_, err = rdb.JSONSet(context.Background(), "room:1", "$", r).Result()
	if err != nil {
		t.Errorf("failed to set json object in redis - %v", err)
	}

	d, err := rdb.JSONGet(ctx, "room:1", "created_at").Result()
	if err != nil {
		t.Errorf("failed to retrieve created_at for room1 - %v", err)
	}

	secs, err := strconv.ParseInt(d, 10, 64)
	if err != nil {
		t.Errorf("failed to parse unix time into seconds - %v", err)
	}
	tim := time.Unix(secs, 0)

	mySkill := 110
	skillDiffAllowed := 20
	thirtySecInterval := time.Second * 30
	if time.Since(tim) > thirtySecInterval*3 {
		skillDiffAllowed += 30
	} else if time.Since(tim) > thirtySecInterval*2 {
		skillDiffAllowed += 20
	} else if time.Since(tim) > thirtySecInterval {
		skillDiffAllowed += 10
	}

	minSkill := max(0, mySkill-skillDiffAllowed)
	maxSkill := mySkill + skillDiffAllowed
	maxPlayerCount := 3 - 1
	fmt.Printf("finding room with skillDiffAllowed=%d, minSkill=%d, maxSkill=%d, maxPlayerCount=%d\n", skillDiffAllowed, minSkill, maxSkill, maxPlayerCount)

	findMatchResult, err := rdb.FTSearch(
		ctx,
		"idx:rooms",
		fmt.Sprintf("@average_skill:[%d %d] @region:%s @player_count:[0,%d]", minSkill, maxSkill, r.Region, maxPlayerCount),
	).Result()
	if err != nil {
		t.Errorf("failed to retrieve query - %v", err)
	}

	fmt.Println(findMatchResult.Docs)
	for _, doc := range findMatchResult.Docs {
		var rm Room
		jsonData := doc.Fields["$"]
		err = json.Unmarshal([]byte(jsonData), &rm)
		if err != nil {
			t.Errorf("failed to read room data - %v", err)
		}
		fmt.Println(rm)
	}
}
