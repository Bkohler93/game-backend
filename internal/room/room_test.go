package room

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/game"
	"github.com/bkohler93/game-backend/internal/utils/redisutils"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
)

func TestRedisRoomDAOMethods(t *testing.T) {
	outerCtx := t.Context()
	var dao *RedisDAO
	var randomRoom Room
	t.Run("create new redis room DAO", func(t *testing.T) {
		client, err := redisutils.NewRedisClient(outerCtx, "localhost:6379", "")
		if err != nil {
			t.Errorf("trouble creating redis client - %v", err)
		}

		dao, err = NewRedisRoomDAO(client)
		if err != nil {
			t.Errorf("trouble initiaizing Redis Room DAO - %v", err)
		}
	})

	t.Run("insert new room", func(t *testing.T) {
		ctx := t.Context()
		randomRoom = RandomRoom()
		err := dao.InsertRoom(ctx, randomRoom)
		if err != nil {
			t.Errorf("error inserting room - %v", err)
		}
	})

	t.Run("get room previously inserted", func(t *testing.T) {
		ctx := t.Context()
		r, err := dao.GetRoom(ctx, randomRoom.RoomId)
		if err != nil {
			t.Errorf("failed to retrieve room with error - %v", err)
		}
		if !reflect.DeepEqual(r, randomRoom) {
			t.Errorf("retrieved room does not equal the room inserted\n--- %v\n---%v\n", r, randomRoom)
		}
	})

	t.Run("create room index", func(t *testing.T) {
		ctx := t.Context()
		err := dao.CreateRoomIndex(ctx)
		if err != nil && err.Error() != "Index already exists" {
			t.Errorf("failed to create index - %v", err)
		}

		info, err := dao.rdb.FTInfo(ctx, RedisRoomIndex).Result()
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
			RoomId:       stringuuid.NewStringUUID(),
			PlayerCount:  1,
			AverageSkill: 100,
			Region:       "na",
			PlayerIds: []stringuuid.StringUUID{
				stringuuid.NewStringUUID(),
			},
			CreatedAt: time.Now().Add(time.Second * 30 * -1).Unix(),
			IsFull:    0,
		}

		r2 := Room{
			RoomId:       stringuuid.NewStringUUID(),
			PlayerCount:  1,
			AverageSkill: 105,
			Region:       "na",
			PlayerIds: []stringuuid.StringUUID{
				stringuuid.NewStringUUID(),
			},
			CreatedAt: time.Now().Add(time.Second * 30 * -2).Unix(),
			IsFull:    0,
		}
		err := dao.CreateRoomIndex(ctx)
		if err != nil && err.Error() != "index already exists" {
			t.Errorf("createRoomIndex should not result in an error. Got - %v", err)
		}

		err = dao.InsertRoom(ctx, r1)
		if err != nil {
			t.Errorf("StoreRoom(r1) should not result in an error. Got - %v", err)
		}
		err = dao.InsertRoom(ctx, r2)
		if err != nil {
			t.Errorf("StoreRoom(r2) should not result in an error. Got - %v", err)
		}

		now := time.Now()
		mySkill := 110
		region := "na"

		minAvgSkill, maxAvgSkill := CalculateMinMaxSkill(mySkill, now)
		maxPlayerCount := 3 - 1

		openRooms, err := dao.QueryOpenRooms(ctx, region, minAvgSkill, maxAvgSkill, maxPlayerCount)
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
		userId := stringuuid.NewStringUUID()
		userSkill := 100
		_, err := dao.QueryOpenRooms(ctx, "na", 90, 120, game.MaxPlayers)
		if err != nil {
			t.Errorf("query rooms resulted in an error - %v", err)
		}

		numPlayers, err := dao.JoinRoom(ctx, randomRoom.RoomId, userId, userSkill)
		if err != nil {
			t.Errorf("unexpected error joining room %s - %v", randomRoom.RoomId, err)
		}
		if numPlayers != 2 {
			t.Errorf("expected 2 players to be in room, got %d", numPlayers)
		}
	})

	t.Run("test add player results in correct new average room skill", func(t *testing.T) {
		ctx := t.Context()
		r := RandomRoom()
		r.AverageSkill = 100

		newUserId := stringuuid.NewStringUUID()
		newUserSkill := 102

		expected := 101

		err := dao.InsertRoom(ctx, r)
		if err != nil {
			t.Errorf("insert room resulted in an error - %v", err)
		}

		playerCount, err := dao.JoinRoom(ctx, r.RoomId, newUserId, newUserSkill)
		if err != nil {
			t.Errorf("join room resulted in an error - %v", err)
		}

		if playerCount != 2 {
			t.Errorf("player count was not updated to 2 - is %v", playerCount)
		}

		returnedRoom, err := dao.GetRoom(ctx, r.RoomId)
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
		room, err := dao.GetRoom(ctx, randomRoom.RoomId)
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
		tm := time.Now().Add(time.Second * -31)

		expected := 30
		actual := CalculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})

	t.Run("test 0 second old time", func(t *testing.T) {
		tm := time.Now()

		expected := BaseSkillThreshold
		actual := CalculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})

	t.Run("test 61 second old time", func(t *testing.T) {
		tm := time.Now().Add(time.Second * -61)

		expected := 50
		actual := CalculateSkillThreshold(tm)

		if expected != actual {
			t.Errorf("expected 31 second old time to evaluate to skill threshold of %d, got %d.", expected, actual)
		}
	})
}

func TestLoadLuaSrc(t *testing.T) {
	t.Run("load AddPlayerToRoom src", func(t *testing.T) {
		_, err := loadLuaSrc(addPlayerToRoomFilePath)
		if err != nil {
			t.Errorf("failed to load AddPlayerToRoom src - %v", err)
		}
	})
}
