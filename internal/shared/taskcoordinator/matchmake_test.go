package taskcoordinator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

func TestMatchmakingRedisStore_PendingTasks(t *testing.T) {
	// ctx := t.Context()

	startup := func(t *testing.T) (store *RedisMatchmakingTaskStore, flush func(), ctx context.Context) {
		ctx = t.Context()
		c, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		store, err = NewRedisMatchmakingTaskStore(c)
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

	t.Run("test adding and getting pending task", func(t *testing.T) {
		store, flushRedis, ctx := startup(t)

		defer flushRedis()

		roomID := uuidstring.NewID()
		now := time.Now().Unix()

		err := store.AddPendingTask(ctx, roomID, now)
		if err != nil {
			t.Errorf("did not expect error when adding new room, member=%s score=%d - %v", roomID, now, err)
		}
		err = store.RemovePendingTask(ctx, roomID)
		if err != nil {
			t.Errorf("did not expect error getting oldest pending task - %v", err)
		}
	})

	t.Run("get pending task when no available tasks should return error", func(t *testing.T) {
		store, flushRedis, ctx := startup(t)
		defer flushRedis()
		roomID := uuidstring.NewID()

		futureScore := time.Now().Add(time.Second * 2).Unix()
		err := store.AddPendingTask(ctx, roomID, futureScore)
		if err != nil {
			t.Errorf("did not expect error when adding task - %v", err)
		}
		rID, err := store.ClaimPendingTask(ctx)
		if err == nil {
			t.Errorf("expected non-nil err, got - %v", err)
		}
		if !errors.Is(err, ErrNoTasksAvailable) {
			t.Errorf("expected no available pending tasks err, got - %v", err)
		}
		if rID != "" {
			t.Errorf("expected id=%s got %s", roomID, rID)
		}
	})

	t.Run("get pending task with time=now should claim task successfully", func(t *testing.T) {
		store, flushRedis, ctx := startup(t)
		defer flushRedis()

		roomID := uuidstring.NewID()
		nowScore := time.Now().Unix()

		err := store.AddPendingTask(ctx, roomID, nowScore)
		if err != nil {
			t.Errorf("did not expect error when adding task - %v", err)
		}
		rID, err := store.ClaimPendingTask(ctx)
		if err != nil {
			t.Errorf("expected no errors, got - %v", err)
		}

		if rID != roomID {
			t.Errorf("expected to claim id=%s got %s", roomID, rID)
		}
	})

	t.Run("get pending task should return the earlier task successfully", func(t *testing.T) {
		store, flushRedis, ctx := startup(t)
		defer flushRedis()

		roomOneID := uuidstring.NewID()
		timeOne := time.Now().Add(time.Second * -2).Unix()

		roomTwoID := uuidstring.NewID()
		timeTwo := time.Now().Add(time.Second * -3).Unix()

		expectedRoomID := roomTwoID

		err := store.AddPendingTask(ctx, roomOneID, timeOne)
		if err != nil {
			t.Errorf("did not expect error when adding task - %v", err)
		}

		err = store.AddPendingTask(ctx, roomTwoID, timeTwo)
		if err != nil {
			t.Errorf("did not expect error when adding task - %v", err)
		}

		rID, err := store.ClaimPendingTask(ctx)
		if err != nil {
			t.Errorf("expected no errors, got - %v", err)
		}

		if rID != expectedRoomID {
			t.Errorf("expected to claim id=%s got %s", expectedRoomID, rID)
		}
	})
}

func TestRedisMatchmakingStore_InProgressTasks(t *testing.T) {
	ctx := t.Context()

	startup := func(t *testing.T) (store *RedisMatchmakingTaskStore, flush func()) {
		c, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		store, err = NewRedisMatchmakingTaskStore(c)
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

	t.Run("test add member to in progress redis store", func(t *testing.T) {
		store, flushRedis := startup(t)
		defer flushRedis()
		roomID := uuidstring.NewID()
		now := time.Now().Unix()

		err := store.AddInProgressTask(ctx, roomID, now)
		if err != nil {
			t.Errorf("expected no error got - %v", err)
		}
	})

	t.Run("test delete member from in progress redis store", func(t *testing.T) {
		store, flushRedis := startup(t)
		defer flushRedis()
		roomID := uuidstring.NewID()
		now := time.Now().Unix()

		err := store.AddInProgressTask(ctx, roomID, now)
		if err != nil {
			t.Errorf("adding member expected no error got - %v", err)
		}
		err = store.RemoveInProgressTask(ctx, roomID)
		if err != nil {
			t.Errorf("removing member expected no error got - %v", err)
		}
	})

	t.Run("add 10 items to in-progress task list starting from now, back 10 seconds. Cutoff is 5 seconds ago.", func(t *testing.T) {
		store, flushRedis := startup(t)
		defer flushRedis()

		expectedIDs := make(map[uuidstring.ID]time.Time)
		allIDs := make(map[uuidstring.ID]time.Time)

		now := time.Now()
		allowableTaskStartTime := now.Add(time.Second * -5) //allow < 5 seconds ago

		for i := 0; i < 10; i++ {
			roomID := uuidstring.NewID()
			tm := now.Add(time.Second * time.Duration(i) * -1)
			err := store.AddInProgressTask(ctx, roomID, tm.Unix())
			if err != nil {
				t.Errorf("did not expect error adding id=%s score=%d - %v", roomID, tm.Unix(), err)
			}
			if tm.Unix() <= allowableTaskStartTime.Unix() { //if true, current roomID would be stale
				expectedIDs[roomID] = tm
			}
			allIDs[roomID] = tm
		}

		actualIDs, err := store.GetStaleInProgressTasks(ctx, allowableTaskStartTime.Unix())
		if err != nil {
			t.Errorf("retrieving stale tasks did not expect an error - %v", err)
		}
		if len(actualIDs) != len(expectedIDs) {
			t.Errorf("did not receive expected number of IDs. expected %d, got %d", len(expectedIDs), len(actualIDs))
		}
		for _, id := range actualIDs {
			if _, ok := expectedIDs[id]; !ok {
				t.Errorf("unexpected id %s with time=%v compared to allowable time=%v", id, allIDs[id], allowableTaskStartTime)
			}
		}
	})
}
