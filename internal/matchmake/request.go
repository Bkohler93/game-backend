package matchmake

type MatchRequest struct {
	UserId string `redis:"user_id" json:"user_id"`
	Name   string `redis:"name"`
	PeerId string `redis:"peer_id" json:"peer_id"`
}
