package uuidstring

import (
	"github.com/google/uuid"
)

type ID string

func NewID() ID {
	return ID(uuid.New().String())
}

func (id ID) UUID() (uuid.UUID, error) {
	if id == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(string(id))
}

func (id ID) String() string {
	return string(id)
}

func (id ID) MarshalBinary() (data []byte, err error) {
	return []byte(id), nil
}
