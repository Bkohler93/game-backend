package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/constants"
	"github.com/bkohler93/game-backend/internal/shared/utils/files"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils/rediskeys"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/redis/go-redis/v9"
)

var (
	ErrDidNotUnlock = errors.New("failed to unlock room")
	ErrDidNotLock   = errors.New("failed to lock room")
)

type Store interface {
	CreateRoomIndex(context.Context) error
	GetRoom(context.Context, uuidstring.ID) (Room, error)
	InsertRoom(context.Context, Room) error
	QueryOpenRooms(ctx context.Context, roomId uuidstring.ID, region string, minAvgSkill, maxAvgSkill, maxPlayerCount int) ([]Room, error)
	JoinRoom(ctx context.Context, roomId uuidstring.ID, userId uuidstring.ID, userSkill int) (Room, error) //returns number of players in room
	LockRoom(ctx context.Context, roomId uuidstring.ID) (uuidstring.ID, error)
	UnlockRoom(ctx context.Context, roomId uuidstring.ID, keyValue uuidstring.ID) error
	CombineRooms(ctx context.Context, room1 Room, room2 Room) (Room, error)
}

type RedisStore struct {
	rdb *redis.Client
	lua map[string]*redis.Script
}

func (store *RedisStore) CombineRooms(ctx context.Context, room1 Room, room2 Room) (Room, error) {
	var rm Room
	keys := []string{rediskeys.RoomsJSONObject(room1.RoomId), rediskeys.RoomsJSONObject(room2.RoomId)}
	var result, err = store.lua[files.LuaCombineRooms].Run(ctx, store.rdb, keys, constants.MaxPlayerCount).Result()
	if err != nil {
		return rm, err
	}
	jsonStr := result.(string)
	err = json.Unmarshal([]byte(jsonStr), &rm)
	if err != nil {
		return rm, errors.New("failed to unmarshal return result into a Room struct")
	}
	return rm, nil
}

func (store *RedisStore) LockRoom(ctx context.Context, roomId uuidstring.ID) (keyValue uuidstring.ID, err error) {
	keyValue = uuidstring.NewID()
	ok, err := store.rdb.SetNX(ctx, rediskeys.RoomLockKey(roomId), keyValue.String(), 10*time.Second).Result()
	if err != nil {
		return keyValue, err
	}
	if !ok {
		return keyValue, ErrDidNotLock
	}
	return keyValue, err
}

func (store *RedisStore) UnlockRoom(ctx context.Context, roomId uuidstring.ID, keyValue uuidstring.ID) error {
	result, err := store.lua[files.LuaUnlockRoom].Run(ctx, store.rdb, []string{rediskeys.RoomLockKey(roomId)}, keyValue.String()).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrDidNotUnlock
	}
	return nil
}

func NewRedisRoomStore(rdb *redis.Client) (*RedisStore, error) {

	luaScripts := make(map[string]*redis.Script)

	addPlayerToRoomSrc, err := files.GetLuaScript(files.LuaAddPlayerToRoom)
	if err != nil {
		return &RedisStore{}, fmt.Errorf("failed to load lua src from '%s' with error - %v", files.LuaAddPlayerToRoom, err)
	}
	luaScripts[files.LuaAddPlayerToRoom] = redis.NewScript(addPlayerToRoomSrc)

	combineRoomsSrc, err := files.GetLuaScript(files.LuaCombineRooms)
	if err != nil {
		return &RedisStore{}, fmt.Errorf("failed to load lua src from '%s' with error - %v", files.LuaCombineRooms, err)
	}
	luaScripts[files.LuaCombineRooms] = redis.NewScript(combineRoomsSrc)

	unlockRoomSrc, err := files.GetLuaScript(files.LuaUnlockRoom)
	if err != nil {
		return &RedisStore{}, fmt.Errorf("failed to load lua src from '%s' with error - %v", files.LuaUnlockRoom, err)
	}
	luaScripts[files.LuaUnlockRoom] = redis.NewScript(unlockRoomSrc)

	return &RedisStore{
		rdb: rdb,
		lua: luaScripts,
	}, nil
}

func (store *RedisStore) GetRoom(ctx context.Context, roomId uuidstring.ID) (Room, error) {
	key := rediskeys.RoomsJSONObject(roomId)
	js, err := store.rdb.JSONGet(ctx, key, "$").Result()
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

func (store *RedisStore) JoinRoom(ctx context.Context, roomId uuidstring.ID, userId uuidstring.ID, userSkill int) (Room, error) {
	var room Room
	key := rediskeys.RoomsJSONObject(roomId)
	result, err := store.lua[files.LuaAddPlayerToRoom].Run(ctx, store.rdb, []string{key}, userId.String(), userSkill, 2).Result() //TODO the '2' magic value should be a constant depending on the game that is being matchmaked more
	if err != nil {
		return room, err
	}
	jsonStr := result.(string)
	err = json.Unmarshal([]byte(jsonStr), &room)
	if err != nil {
		return room, errors.New("failed to unmarshal return result into a Room struct")
	}
	return room, nil
}

const (
	RedisRoomIndex = "idx:rooms"
)

func (store *RedisStore) InsertRoom(ctx context.Context, r Room) error {
	key := rediskeys.RoomsJSONObject(r.RoomId)
	return store.rdb.JSONSet(ctx, key, "$", r).Err()
}

func (store *RedisStore) CreateRoomIndex(ctx context.Context) error {
	indices, err := store.rdb.FT_List(ctx).Result()
	if err != nil {
		return fmt.Errorf("error retrieving indices: %v", err)
	}

	if slices.Contains(indices, RedisRoomIndex) {
		return errors.New("index already exists")
	}

	return store.rdb.FTCreate(
		ctx,
		RedisRoomIndex,
		// Options:
		&redis.FTCreateOptions{
			OnJSON: true,
			Prefix: []interface{}{"rooms:"},
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

func (store *RedisStore) QueryOpenRooms(ctx context.Context, roomId uuidstring.ID, region string, minAvgSkill, maxAvgSkill, maxPlayerCount int) ([]Room, error) {
	var rooms []Room
	findMatchResult, err := store.rdb.FTSearch(
		ctx,
		"idx:rooms",
		fmt.Sprintf("@average_skill:[%d %d] @region:%s @player_count:[0,%d]", minAvgSkill, maxAvgSkill, region, maxPlayerCount),
	).Result()
	if err != nil {
		return rooms, err
	}

	rooms, err = redisutils.SliceFromRedisDocs[Room](findMatchResult.Docs)
	if err != nil {
		return rooms, fmt.Errorf("error creating slice of models.Room from redis Query result - %v", err)
	}
	filtered := rooms[:0]
	for _, rm := range rooms {
		if rm.RoomId != roomId {
			filtered = append(filtered, rm)
		}
	}
	return filtered, err
}
