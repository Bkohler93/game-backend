package routingtable

import (
	"errors"
	"testing"
	"time"

	"github.com/bkohler93/game-backend/internal/shared/utils/redisutils"
)

func TestRoutingTable_AddGet(t *testing.T) {
	t.Run("add/get valid entries", func(t *testing.T) {
		redisClient, err := redisutils.NewRedisMatchmakeClient(t.Context())
		if err != nil {
			panic(err)
		}
		rt := NewRoutingTable("test", redisClient)
		ids := []string{"client-1", "client-2"}
		hosts := []string{"host-1", "host-3"}
		for i, id := range ids {
			err = rt.Add(t.Context(), id, hosts[i])
			if err != nil {
				t.Fatalf("unexpected error adding valid ids and hosts - %v", err)
			}
		}

		for i, id := range ids {
			host, err := rt.Get(t.Context(), id)
			if err != nil {
				t.Fatalf("unexpected error retrieving id[%s] host - %v", id, err)
			}
			if host != hosts[i] {
				t.Fatalf("retrieved host[%s] for id[%s] not equal to expected host[%s]", host, id, hosts[i])
			}
		}
	})

	t.Run("test redis key invalidation", func(t *testing.T) {
		ctx := t.Context()
		redisClient, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		rt := NewRoutingTable("test", redisClient)
		id := "client1"
		host := "host1"
		err = rt.Add(ctx, id, host)
		if err != nil {
			t.Fatalf("did not expect error adding id[%s] host[%s] - %v", id, host, err)
		}
		time.Sleep(RouteTTL)
		retrievedHost, err := rt.Get(ctx, id)
		if !errors.Is(err, ErrKeyExpired) {
			t.Fatalf("expected err[%v] got err[%v]", ErrKeyExpired, err)
		}
		if retrievedHost != "" {
			t.Fatalf("expected empty string got host[%s]", retrievedHost)
		}
	})

	t.Run("test local cache invalidation", func(t *testing.T) {
		ctx := t.Context()
		redisClient, err := redisutils.NewRedisMatchmakeClient(ctx)
		if err != nil {
			panic(err)
		}
		rt := NewRoutingTable("test", redisClient)
		id := "client1"
		host := "host1"
		err = rt.Add(ctx, id, host)
		if err != nil {
			t.Fatalf("did not expect error adding id[%s] host[%s] - %v", id, host, err)
		}
		retrievedHost, err := rt.Get(ctx, id)
		if err != nil {
			t.Fatalf("failed to retrieve host[%s] for id[%s] - err[%v]", host, id, err)
		}
		newHost := "host2"
		err = rt.Add(ctx, id, newHost)
		if err != nil {
			t.Fatalf("did not expect error adding id[%s] host[%s] - %v", id, newHost, err)
		}
		time.Sleep(time.Second * 2)
		retrievedHost, err = rt.Get(ctx, id)
		if err != nil {
			t.Fatalf("failed to retrieve host[%s] for id[%s] - err[%v]", host, id, err)
		}
		if retrievedHost != host {
			t.Fatalf("expected to retrieve original host[%s] - got host[%s]", host, retrievedHost)
		}
		time.Sleep(RouteTTL - (time.Second * 2))
		retrievedHost, err = rt.Get(ctx, id)
		if !errors.Is(err, ErrKeyExpired) {
			t.Fatalf("expected to receive err[%v] - got err[%v]", ErrKeyExpired, err)
		}
		if retrievedHost != "" {
			t.Fatalf("expected to retrieve empty string - got host[%s]", retrievedHost)
		}

		// adding a new host value should force the RouteTable to retrieve most up to date value
		err = rt.Add(ctx, id, newHost)
		if err != nil {
			t.Fatalf("did not expect error adding id[%s] host[%s] - %v", id, newHost, err)
		}
		retrievedHost, err = rt.Get(ctx, id)
		if err != nil {
			t.Fatalf("failed to retrieve host[%s] for id[%s] - err[%v]", host, id, err)
		}
		if retrievedHost != newHost {
			t.Fatalf("expected to receive newHost[%v] - got host[%v]", newHost, retrievedHost)
		}
	})
}

func TestRoutingTable_RefreshTTL(t *testing.T) {
	redisClient, err := redisutils.NewRedisMatchmakeClient(t.Context())
	if err != nil {
		panic(err)
	}
	ctx := t.Context()
	rt := NewRoutingTable("test", redisClient)
	id := "client1"
	host := "host1"
	err = rt.Add(ctx, id, host)
	if err != nil {
		t.Fatalf("error adding id[%s] host[%s] - %v", id, host, err)
	}
	time.Sleep(RouteTTL - time.Second)
	err = rt.RefreshTTL(ctx, id)
	if err != nil {
		t.Fatalf("error updating id[%s] TTL - %v", id, err)
	}
	time.Sleep(time.Second * 2)
	retrievedHost, err := rt.Get(ctx, id)
	if err != nil {
		t.Fatalf("error retrieving id[%s] - %v", id, err)
	}
	if retrievedHost != host {
		t.Fatalf("retrieving host[%s] - expected host[%s]", retrievedHost, host)
	}
}
