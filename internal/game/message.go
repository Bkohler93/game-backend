package game

import "github.com/bkohler93/game-backend/pkg/stringuuid"

type ClientActionBase struct {
	GameID   stringuuid.StringUUID
	ClientID stringuuid.StringUUID
}

type ServerResponse struct{}
