package metadata

const (
	Game         = "game"
	Remain       = ""
	Play         = "play"
	GameOver     = "game_over"
	Matchmake    = "matchmake"
	TransitionTo = "transition_to"
	RoomID       = "room_id"
	MetaDataKey  = "meta_data"
)

type MetaData map[string]string
