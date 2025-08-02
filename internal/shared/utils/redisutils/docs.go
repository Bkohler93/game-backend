package redisutils

import (
	"encoding/json"
	"slices"

	"github.com/redis/go-redis/v9"
)

func SliceFromRedisDocs[T any](docs []redis.Document) ([]T, error) {
	items := []T{}
	for _, doc := range docs {
		var item T
		jsonData := doc.Fields["$"]
		err := json.Unmarshal([]byte(jsonData), &item)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, nil
}

func SortedSliceFromRedisDocs[T any](docs []redis.Document, sortFunc func(a, b T) int) ([]T, error) {
	items := []T{}
	for _, doc := range docs {
		var item T
		jsonData := doc.Fields["$"]
		err := json.Unmarshal([]byte(jsonData), &item)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	slices.SortFunc(items, sortFunc)
	return items, nil
}
