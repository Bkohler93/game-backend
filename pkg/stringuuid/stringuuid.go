package stringuuid

import (
	"github.com/google/uuid"
)

type UserId string

func NewUserId() UserId {
	return UserId(uuid.New().String())
}

func (id UserId) UUID() uuid.UUID {
	if id == "" {
		return uuid.Nil
	}
	return uuid.MustParse(string(id))
}

func (id UserId) String() string {
	return string(id)
}

func (u UserId) MarshalBinary() (data []byte, err error) {
	return []byte(u.String()), nil
}
