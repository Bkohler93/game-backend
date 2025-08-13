package redisutils

import (
	"encoding/json"
	"errors"
)

var (
	ErrInvalidJsonData = errors.New("supplied data was not a json encoded string")
)

func JsonTo[T any](jsonData any) (T, error) {
	var t T
	jsonStr, ok := jsonData.(string)
	if !ok {
		return t, ErrInvalidJsonData
	}
	err := json.Unmarshal([]byte(jsonStr), &t)
	if err != nil {
		return t, ErrInvalidJsonData
	}
	return t, err
}
