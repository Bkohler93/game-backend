package taskcoordinator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

func TestGameRedisStore(t *testing.T) {
	startup := func(t *testing.T) (store *RedisGameTaskStore, flush func(), ctx context.Context) {
		ctx = t.Context()
		c, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		store, err = NewRedisGameTaskStore(c)
		if err != nil {
			panic(err)
		}
		flush = func() {
			err := store.rdb.FlushDB(ctx).Err()
			if err != nil {
				panic(err)
			}
		}
		return
	}

	t.Run("test adding and getting new game task", func(t *testing.T) {
		store, flushRedis, ctx := startup(t)

		defer flushRedis()

		roomID := uuidstring.NewID()
		now := time.Now().Unix()

		err := store.AddNewGameTask(ctx, roomID)
		if err != nil {
			t.Errorf("did not expect error when adding new game task, member=%s score=%d - %v", roomID, now, err)
		}
		testValue, err := store.RemoveOldestNewGameTask(ctx)
		if err != nil {
			t.Errorf("did not expect error getting oldest pending task - %v", err)
		}
		if testValue != roomID {
			t.Errorf("expected %s to be %s", testValue, roomID)
		}
	})

	t.Run("get new game task when no available tasks should return error", func(t *testing.T) {
		store, flushRedis, ctx := startup(t)
		defer flushRedis()

		rID, err := store.RemoveOldestNewGameTask(ctx)
		if err == nil {
			t.Errorf("expected non-nil err, got - %v", err)
		}
		if !errors.Is(err, ErrNoTasksAvailable) {
			t.Errorf("expected no available pending tasks err, got - %v", err)
		}
		if rID != "" {
			t.Errorf("expected no id got %s", rID)
		}
	})
}
