package metadata

const (
	Setup        = "setup"
	Remain       = ""
	Play         = "play"
	GameOver     = "game_over"
	Matchmake    = "matchmake"
	NewGameState = "new_game_state"
	RoomID       = "room_id"
	MetaDataKey  = "meta_data"
)

type MetaData map[string]string
