package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/bkohler93/game-backend/internal/game"
	"github.com/bkohler93/game-backend/internal/utils/redisutils"
	"github.com/bkohler93/game-backend/pkg/stringuuid"
	"github.com/redis/go-redis/v9"
)

var (
	luaScriptBasePath       = "../../db/redis/scripts"
	addPlayerToRoomFilePath = fmt.Sprintf("%s/add_player_to_room.lua", luaScriptBasePath)
)

type DAO interface {
	CreateRoomIndex(context.Context) error
	GetRoom(context.Context, stringuuid.StringUUID) (Room, error)
	InsertRoom(context.Context, Room) error
	QueryOpenRooms(ctx context.Context, region string, minAvgSkill, maxAvgSkill, maxPlayerCount int) ([]Room, error)
	JoinRoom(roomId stringuuid.StringUUID, userId stringuuid.StringUUID) (int, error) //returns number of players in room
}

type RedisDAO struct {
	rdb *redis.Client
	lua map[string]*redis.Script
}

func (dao *RedisDAO) GetRoom(ctx context.Context, roomId stringuuid.StringUUID) (Room, error) {
	js, err := dao.rdb.JSONGet(ctx, "room:"+roomId.String(), "$").Result()
	if err != nil {
		return Room{}, err
	}
	var rooms []Room
	err = json.Unmarshal([]byte(js), &rooms)
	if err != nil {
		return Room{}, err
	}
	return rooms[0], nil
}

func (dao *RedisDAO) JoinRoom(ctx context.Context, roomId stringuuid.StringUUID, userId stringuuid.StringUUID, userSkill int) (int, error) {
	var newCount int
	key := fmt.Sprintf("room:%s", roomId)
	result, err := dao.lua[addPlayerToRoomFilePath].Run(ctx, dao.rdb, []string{key}, userId.String(), userSkill, game.MaxPlayers, redisutils.MatchmakingRoomEventsStream()).Result()
	if err != nil {
		return newCount, err
	}
	newCount64, ok := result.(int64)
	if !ok {
		return int(newCount64), errors.New("failed to cast script return result into an int")
	}

	return int(newCount64), nil
}

const (
	RedisRoomIndex = "idx:rooms"
)

func NewRedisRoomDAO(rdb *redis.Client) (*RedisDAO, error) {
	luaScripts := make(map[string]*redis.Script)
	addPlayerToRoomSrc, err := loadLuaSrc(addPlayerToRoomFilePath)
	if err != nil {
		return &RedisDAO{}, fmt.Errorf("failed to load lua src from '%s' with error - %v", addPlayerToRoomFilePath, err)
	}
	luaScripts[addPlayerToRoomFilePath] = redis.NewScript(addPlayerToRoomSrc)

	return &RedisDAO{
		rdb: rdb,
		lua: luaScripts,
	}, nil
}

func loadLuaSrc(filePath string) (string, error) {
	bytes, err := os.ReadFile(filePath)
	return string(bytes), err
}

func (dao *RedisDAO) InsertRoom(ctx context.Context, r Room) error {
	key := fmt.Sprintf("room:%s", r.RoomId)
	return dao.rdb.JSONSet(ctx, key, "$", r).Err()
}

func (dao *RedisDAO) CreateRoomIndex(ctx context.Context) error {
	indices, err := dao.rdb.FT_List(ctx).Result()
	if err != nil {
		return fmt.Errorf("error retrieving indices: %v", err)
	}

	if slices.Contains(indices, RedisRoomIndex) {
		return errors.New("index already exists")
	}

	return dao.rdb.FTCreate(
		ctx,
		RedisRoomIndex,
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
		&redis.FieldSchema{
			FieldName: "$.is_full",
			As:        "is_full",
			FieldType: redis.SearchFieldTypeNumeric,
		},
	).Err()
}

func (dao *RedisDAO) QueryOpenRooms(ctx context.Context, region string, minAvgSkill, maxAvgSkill, maxPlayerCount int) ([]Room, error) {
	var rooms []Room
	findMatchResult, err := dao.rdb.FTSearch(
		ctx,
		"idx:rooms",
		fmt.Sprintf("@average_skill:[%d %d] @region:%s @player_count:[0,%d]", minAvgSkill, maxAvgSkill, region, maxPlayerCount),
	).Result()
	if err != nil {
		return rooms, fmt.Errorf("error retrieving open rooms from redis - %v", err)
	}

	rooms, err = redisutils.SliceFromRedisDocs[Room](findMatchResult.Docs)
	if err != nil {
		return rooms, fmt.Errorf("error creating slice of models.Room from redis Query result - %v", err)
	}
	return rooms, err
}
