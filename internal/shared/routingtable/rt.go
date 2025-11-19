package routingtable

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrNoHostEntry = errors.New("no host is found for that identifier")
	ErrKeyNotFound = errors.New("the key was not found in redis")
	ErrKeyExpired  = errors.New("key expired or was never created")
)

func unknownError(err error) error {
	return fmt.Errorf("received unknown err - %v", err)
}

const (
	InvalidateCacheAfter = time.Second * 5  // used by Consumer to force redis.Get for up-to-date host
	RouteTTL             = time.Second * 15 // used by Manager to keep redis value active
)

type RouteManager interface {
	Add(ctx context.Context, identifiers []string, hosts []string) error
	Delete(id string) error     // on graceful shutdown delete key immediately
	RefreshTTL(id string) error // maybe during ping pong with client refresh this, for game do it during timer
}

type RouteConsumer interface {
	Get(id string) (string, error)
}

type RoutingTable struct {
	rdb         *redis.Client
	routePrefix string
	cache       map[string]cachedEntry
}

type cachedEntry struct {
	ttl  int64
	host string
}

func NewRoutingTable(routePrefix string, rdb *redis.Client) *RoutingTable {
	return &RoutingTable{
		rdb:         rdb,
		routePrefix: routePrefix,
		cache:       make(map[string]cachedEntry),
	}
}

func (rt *RoutingTable) key(id string) string {
	return fmt.Sprintf("%s_route:%s", rt.routePrefix, id)
}

func (rt *RoutingTable) RefreshTTL(ctx context.Context, id string) error {
	didSet, err := rt.rdb.Expire(ctx, rt.key(id), RouteTTL).Result()
	if err != nil {
		return unknownError(err)
	}
	if !didSet {
		return ErrKeyNotFound
	}
	return nil
}

func (rt *RoutingTable) Add(ctx context.Context, identifier string, host string) error {
	err := rt.rdb.Set(ctx, rt.key(identifier), host, RouteTTL).Err()
	if err != nil {
		return unknownError(err)
	}
	return nil
}

func (rt *RoutingTable) Get(ctx context.Context, identifier string) (string, error) {
	key := rt.key(identifier)
	if entry, ok := rt.cache[key]; ok {
		if time.Since(time.Unix(entry.ttl, 0)) < InvalidateCacheAfter {
			return entry.host, nil
		}
		delete(rt.cache, identifier)
	}
	res, err := rt.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrKeyExpired
		}
		return "", unknownError(err)
	}
	if res == "" {
		return "", ErrNoHostEntry
	}

	rt.cache[key] = cachedEntry{
		ttl:  time.Now().Unix(),
		host: res,
	}
	return res, nil
}
