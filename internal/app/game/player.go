package game

import "github.com/bkohler93/game-backend/pkg/uuidstring"

type Player struct {
	UserID uuidstring.ID
	Name   string
}
