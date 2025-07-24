package stringuuid

import (
	"github.com/google/uuid"
)

type StringUUID string

func NewStringUUID() StringUUID {
	return StringUUID(uuid.New().String())
}

func (id StringUUID) UUID() uuid.UUID {
	if id == "" {
		return uuid.Nil
	}
	return uuid.MustParse(string(id))
}

func (id StringUUID) String() string {
	return string(id)
}

func (u StringUUID) MarshalBinary() (data []byte, err error) {
	return []byte(u.String()), nil
}
