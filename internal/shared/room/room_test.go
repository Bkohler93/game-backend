package room

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/app/game"
	"github.com/bkohler93/game-backend/internal/shared/utils/files"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

func TestNewRedisRoomStoreMethods(t *testing.T) {
	outerCtx := t.Context()
	var store *RedisStore
	var randomRoom Room

	t.Cleanup(func() {
		ctx := context.Background()
		err := store.rdb.FlushDB(ctx).Err()
		if err != nil {
			panic("failed to flush redis db - " + err.Error())
		}
	})
	t.Run("create new redis room Store", func(t *testing.T) {
		client, err := redisutils.NewRedisClient(outerCtx)
		if err != nil {
			t.Errorf("trouble creating redis client - %v", err)
		}

		store, err = NewRedisRoomStore(client)
		if err != nil {
			t.Errorf("trouble initiaizing Redis Room Store - %v", err)
		}
	})

	t.Run("insert new room", func(t *testing.T) {
		ctx := t.Context()
		randomRoom = RandomRoom()
		err := store.InsertRoom(ctx, randomRoom)
		if err != nil {
			t.Errorf("error inserting room - %v", err)
		}
	})

	t.Run("get room previously inserted", func(t *testing.T) {
		ctx := t.Context()
		r, err := store.GetRoom(ctx, randomRoom.RoomId)
		if err != nil {
			t.Errorf("failed to retrieve room with error - %v", err)
		}
		if !reflect.DeepEqual(r, randomRoom) {
			t.Errorf("retrieved room does not equal the room inserted\n--- %v\n---%v\n", r, randomRoom)
		}
	})

	t.Run("create room index", func(t *testing.T) {
		ctx := t.Context()
		err := store.CreateRoomIndex(ctx)
		if err != nil && err.Error() != "Index already exists" {
			t.Errorf("failed to create index - %v", err)
		}

		info, err := store.rdb.FTInfo(ctx, RedisRoomIndex).Result()
		if err != nil {
			t.Errorf("should not result in an error when retrieving index info - %v", err)
		}
		if info.IndexName != RedisRoomIndex {
			t.Errorf("index info name should be %s, got %s", RedisRoomIndex, info.IndexName)
		}
	})

	t.Run("store and query rooms", func(t *testing.T) {
		ctx := t.Context()

		r1 := Room{
			RoomId:       uuidstring.NewID(),
			PlayerCount:  1,
			AverageSkill: 100,
			Region:       "na",
			PlayerIds: []uuidstring.ID{
				uuidstring.NewID(),
			},
			CreatedAt: time.Now().Add(time.Second * 30 * -1).Unix(),
			IsFull:    0,
		}

		r2 := Room{
			RoomId:       uuidstring.NewID(),
			PlayerCount:  1,
			AverageSkill: 105,
			Region:       "na",
			PlayerIds: []uuidstring.ID{
				uuidstring.NewID(),
			},
			CreatedAt: time.Now().Add(time.Second * 30 * -2).Unix(),
			IsFull:    0,
		}
		err := store.CreateRoomIndex(ctx)
		if err != nil && err.Error() != "index already exists" {
			t.Errorf("createRoomIndex should not result in an error. Got - %v", err)
		}

		err = store.InsertRoom(ctx, r1)
		if err != nil {
			t.Errorf("StoreRoom(r1) should not result in an error. Got - %v", err)
		}
		err = store.InsertRoom(ctx, r2)
		if err != nil {
			t.Errorf("StoreRoom(r2) should not result in an error. Got - %v", err)
		}

		now := time.Now().Unix()
		mySkill := 110
		region := "na"

		minAvgSkill, maxAvgSkill := CalculateMinMaxSkill(mySkill, now)
		maxPlayerCount := 3 - 1

		openRooms, err := store.QueryOpenRooms(ctx, region, minAvgSkill, maxAvgSkill, maxPlayerCount)
		if err != nil {
			t.Errorf("failed to retrieve open rooms in query - %v", err)
		}
		slices.SortStableFunc(openRooms, SortedByCreatedAtFunc)

		if openRooms[0].AverageSkill < openRooms[1].AverageSkill {
			t.Errorf("incorrect order of rooms... rooms[0].createdAt={%s}, rooms[1].createdAt={%s}", time.Unix(openRooms[0].CreatedAt, 0).String(), time.Unix(openRooms[1].CreatedAt, 0).String())
		}
	})

	t.Run("test add player to room", func(t *testing.T) {
		ctx := t.Context()
		userId := uuidstring.NewID()
		userSkill := 100
		_, err := store.QueryOpenRooms(ctx, "na", 90, 120, game.MaxPlayers)
		if err != nil {
			t.Errorf("query rooms resulted in an error - %v", err)
		}

		room, err := store.JoinRoom(ctx, randomRoom.RoomId, userId, userSkill)
		if err != nil {
			t.Errorf("unexpected error joining room %s - %v", randomRoom.RoomId, err)
		}
		if room.PlayerCount != 2 {
			t.Errorf("expected 2 players to be in room, got %d", room.PlayerCount)
		}
	})

	t.Run("test add player results in correct new average room skill", func(t *testing.T) {
		ctx := t.Context()
		r := RandomRoom()
		r.AverageSkill = 100

		newUserId := uuidstring.NewID()
		newUserSkill := 102

		expected := 101

		err := store.InsertRoom(ctx, r)
		if err != nil {
			t.Errorf("insert room resulted in an error - %v", err)
		}

		room, err := store.JoinRoom(ctx, r.RoomId, newUserId, newUserSkill)
		if err != nil {
			t.Errorf("join room resulted in an error - %v", err)
		}

		if room.PlayerCount != 2 {
			t.Errorf("player count was not updated to 2 - is %v", room.PlayerCount)
		}

		returnedRoom, err := store.GetRoom(ctx, r.RoomId)
		if err != nil {
			t.Errorf("failed to get room id=%s - %v", r.RoomId, err)
		}
		if returnedRoom.AverageSkill != expected {
			t.Errorf("expected average skill to be %d, got %d", expected, returnedRoom.AverageSkill)
		}
	})

	//TODO also need to test the stream
	t.Run("test add player to finish filling a room", func(t *testing.T) {
		ctx := t.Context()
		room, err := store.GetRoom(ctx, randomRoom.RoomId)
		if err != nil {
			t.Errorf("failed to get room with error - %v", err)
		}
		playerCount := room.PlayerCount
		if room.IsFull != 1 {
			t.Errorf("expected room to be full. is not even though player count is %d", playerCount)
		}
	})
}

func TestCalculateThreshold(t *testing.T) {
	t.Run("test 31 second old time", func(t *testing.T) {
		tm := time.Now().Add(time.Second * -31).Unix()

		expected := 30
		actual := CalculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})

	t.Run("test 0 second old time", func(t *testing.T) {
		tm := time.Now().Unix()

		expected := BaseSkillThreshold
		actual := CalculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})

	t.Run("test 61 second old time", func(t *testing.T) {
		tm := time.Now().Add(time.Second * -61).Unix()

		expected := 50
		actual := CalculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})
}

func TestLoadLuaSrc(t *testing.T) {
	t.Run("load AddPlayerToRoom src", func(t *testing.T) {
		_, err := files.GetLuaScript(files.LuaAddPlayerToRoom)
		if err != nil {
			t.Errorf("failed to load AddPlayerToRoom src - %v", err)
		}
	})
}
