package metadata

type MetaDataKey int

const (
	TransitionTo MetaDataKey = iota
	RoomIDKey
	MsgIdKey
)

type MetaDataValue string

const (
	Game      MetaDataValue = "game"
	Remain    MetaDataValue = ""
	Play      MetaDataValue = "play"
	GameOver  MetaDataValue = "game_over"
	Matchmake MetaDataValue = "matchmake"
)

type MetaData map[MetaDataKey]MetaDataValue
