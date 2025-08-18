package matchmake

import (
	"github.com/bkohler93/game-backend/internal/shared/message"
	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

type MatchmakingClientMessage interface {
	message.Message
}

func UnmarshalMatchmakingClientMessage(data []byte) (MatchmakingClientMessage, error) {
	return message.UnmarshalWrappedType[MatchmakingClientMessage](data, matchmakingMessageRegistry)
	//var temp struct {
	//	TypeDiscriminator string `json:"$type"`
	//}
	//if err := json.Unmarshal(data, &temp); err != nil {
	//	return nil, err
	//}
	//
	//if constructor, ok := matchmakingMessageRegistry[temp.TypeDiscriminator]; ok {
	//	concreteMessage := constructor()
	//	if err := json.Unmarshal(data, &concreteMessage); err != nil {
	//		return nil, err
	//	}
	//	return concreteMessage, nil
	//}
	//
	//return nil, fmt.Errorf("unknown matchmaking message type: %s", temp.TypeDiscriminator)
}

var matchmakingMessageRegistry = map[string]func() MatchmakingClientMessage{
	string(PlayerLeftRoom):   func() MatchmakingClientMessage { return &PlayerLeftRoomMessage{} },
	string(PlayerJoinedRoom): func() MatchmakingClientMessage { return &PlayerJoinedRoomMessage{} },
	string(RoomChanged):      func() MatchmakingClientMessage { return &RoomChangedMessage{} },
}

type MatchmakingClientMessageType string

const (
	PlayerLeftRoom   MatchmakingClientMessageType = "PlayerLeftRoom"
	PlayerJoinedRoom MatchmakingClientMessageType = "PlayerJoinedRoom"
	RoomChanged      MatchmakingClientMessageType = "RoomChanged"
	RoomFull         MatchmakingClientMessageType = "RoomFull"
)

type RoomFullMessage struct {
	TypeDiscriminator string        `json:"$type"`
	RoomID            uuidstring.ID `json:"room_id"`
	PlayerCount       int           `json:"player_count"`
}

//func (r *RoomFullMessage) GetMetaData() map[string]interface{} {
//	return r.metaData
//}
//
//func (r *RoomFullMessage) SetMetaData(m map[string]interface{}) {
//	r.metaData = m
//}
//
//func (r *RoomFullMessage) Ack(ctx context.Context) error {
//	return r.ack(ctx)
//}
//
//func (r *RoomFullMessage) SetAck(f func(context.Context) error) {
//	r.ack = f
//}

func NewRoomFullMessage(roomID uuidstring.ID, playerCount int) *RoomFullMessage {
	return &RoomFullMessage{
		TypeDiscriminator: string(RoomFull),
		RoomID:            roomID,
		PlayerCount:       playerCount,
	}
}

func (r *RoomFullMessage) GetDiscriminator() string {
	return r.TypeDiscriminator
}

//func (r *RoomFullMessage) GetID() string {
//	return r.ID
//}
//
//func (r *RoomFullMessage) SetID(s string) {
//	r.ID = s
//}

type PlayerLeftRoomMessage struct {
	TypeDiscriminator string        `json:"$type"`
	UserLeftId        uuidstring.ID `json:"user_left_id"`
}

//func (p *PlayerLeftRoomMessage) GetMetaData() map[string]interface{} {
//	return p.metaData
//}
//
//func (p *PlayerLeftRoomMessage) SetMetaData(m map[string]interface{}) {
//	p.metaData = m
//}
//
//func (p *PlayerLeftRoomMessage) Ack(ctx context.Context) error {
//	return p.ack(ctx)
//}
//
//func (p *PlayerLeftRoomMessage) SetAck(f func(context.Context) error) {
//	p.ack = f
//}

//func (p *PlayerLeftRoomMessage) GetID() string {
//	return p.ID
//}
//
//func (p *PlayerLeftRoomMessage) SetID(s string) {
//	p.ID = s
//}

func (p *PlayerLeftRoomMessage) GetDiscriminator() string {
	return p.TypeDiscriminator
}

func NewPlayerLeftRoomMessage(userLeftId uuidstring.ID) PlayerLeftRoomMessage {
	return PlayerLeftRoomMessage{
		TypeDiscriminator: string(PlayerLeftRoom),
		UserLeftId:        userLeftId,
	}
}

type PlayerJoinedRoomMessage struct {
	TypeDiscriminator string        `json:"$type"`
	UserJoinedId      uuidstring.ID `json:"user_joined_id"`
}

//func (p *PlayerJoinedRoomMessage) GetMetaData() map[string]interface{} {
//	return p.metaData
//}
//
//func (p *PlayerJoinedRoomMessage) SetMetaData(m map[string]interface{}) {
//	p.metaData = m
//}
//
//func (p *PlayerJoinedRoomMessage) Ack(ctx context.Context) error {
//	return p.ack(ctx)
//}
//
//func (p *PlayerJoinedRoomMessage) SetAck(f func(context.Context) error) {
//	p.ack = f
//}

//func (p *PlayerJoinedRoomMessage) GetID() string {
//	return p.ID
//}
//
//func (p *PlayerJoinedRoomMessage) SetID(s string) {
//	p.ID = s
//}

func (p *PlayerJoinedRoomMessage) GetDiscriminator() string {
	return p.TypeDiscriminator
}

func NewPlayerJoinedRoomMessage(userJoinedId uuidstring.ID) *PlayerJoinedRoomMessage {
	return &PlayerJoinedRoomMessage{
		TypeDiscriminator: "PlayerJoinedRoom",
		UserJoinedId:      userJoinedId,
	}
}

type RoomChangedMessage struct {
	TypeDiscriminator string        `json:"$type"`
	NewRoomId         uuidstring.ID `json:"new_room_id"`
	PlayerCount       int           `json:"player_count"`
	AvgSkill          int           `json:"avg_skill"`
}

//func (r *RoomChangedMessage) GetMetaData() map[string]interface{} {
//	return r.MetaData
//}
//
//func (r *RoomChangedMessage) SetMetaData(m map[string]interface{}) {
//	r.MetaData = m
//}
//
//func (r *RoomChangedMessage) Ack(ctx context.Context) error {
//	return r.ack(ctx)
//}
//
//func (r *RoomChangedMessage) SetAck(f func(context.Context) error) {
//	r.ack = f
//}

//func (r *RoomChangedMessage) GetID() string {
//	return r.ID
//}
//
//func (r *RoomChangedMessage) SetID(s string) {
//	r.ID = s
//}

func (r *RoomChangedMessage) GetDiscriminator() string {
	return r.TypeDiscriminator
}

func NewRoomChangedMessage(newRoomId uuidstring.ID, playerCount, avgSkill int) *RoomChangedMessage {
	return &RoomChangedMessage{
		TypeDiscriminator: "RoomChanged",
		NewRoomId:         newRoomId,
		PlayerCount:       playerCount,
		AvgSkill:          avgSkill,
	}
}

type MatchmakingServerMessage interface {
	message.Message
}

func UnmarshalMatchmakingServerMessage(data []byte) (MatchmakingServerMessage, error) {
	return message.UnmarshalWrappedType[MatchmakingServerMessage](data, serverMessageTypeRegistry)
	//var temp struct {
	//	TypeDiscriminator string `json:"$type"`
	//}
	//if err := json.Unmarshal(data, &temp); err != nil {
	//	return nil, err
	//}
	//
	//if constructor, ok := serverMessageTypeRegistry[temp.TypeDiscriminator]; ok {
	//	concreteMessage := constructor()
	//	if err := json.Unmarshal(data, &concreteMessage); err != nil {
	//		return nil, err
	//	}
	//	return concreteMessage, nil
	//}
	//
	//return nil, fmt.Errorf("unknown matchmaking message type: %s", temp.TypeDiscriminator)
}

var serverMessageTypeRegistry = map[string]func() MatchmakingServerMessage{
	string(RequestMatchmaking): func() MatchmakingServerMessage { return &RequestMatchmakingMessage{} },
	string(ExitMatchmaking):    func() MatchmakingServerMessage { return &ExitMatchmakingMessage{} },
}

type MatchmakingServerMessageType string

const (
	RequestMatchmaking MatchmakingServerMessageType = "RequestMatchmaking"
	ExitMatchmaking    MatchmakingServerMessageType = "ExitMatchmaking"
)

type RequestMatchmakingMessage struct {
	TypeDiscriminator string        `json:"$type"`
	UserId            uuidstring.ID `redis:"user_id" json:"user_id"`
	Name              string        `redis:"name" json:"name"`
	TimeCreated       int64         `redis:"time_created" json:"time_created"` //TODO remove this? The Room object will contain a Retry count
	Skill             int           `redis:"skill" json:"skill"`
	Region            string        `redis:"region" json:"region"`
}

//func (m *RequestMatchmakingMessage) GetMetaData() map[string]interface{} {
//	return m.metaData
//}
//
//func (m *RequestMatchmakingMessage) SetMetaData(md map[string]interface{}) {
//	m.metaData = md
//}
//
//func (m *RequestMatchmakingMessage) Ack(ctx context.Context) error {
//	return m.ack(ctx)
//}
//
//func (m *RequestMatchmakingMessage) SetAck(f func(context.Context) error) {
//	m.ack = f
//}

//func (m *RequestMatchmakingMessage) GetID() string {
//	return m.ID
//}
//
//func (m *RequestMatchmakingMessage) SetID(s string) {
//	m.ID = s
//}

func (m *RequestMatchmakingMessage) GetDiscriminator() string {
	return m.TypeDiscriminator
}

func (m *RequestMatchmakingMessage) Equals(other RequestMatchmakingMessage) bool {
	return m.UserId == other.UserId && m.Region == other.Region && m.Name == other.Name && m.Skill == other.Skill && m.TimeCreated == other.TimeCreated
}

func NewRequestMatchmakingMessage(userId uuidstring.ID, name string, timeCreated int64, skill int, region string) RequestMatchmakingMessage {
	return RequestMatchmakingMessage{
		TypeDiscriminator: string(RequestMatchmaking),
		UserId:            userId,
		Name:              name,
		TimeCreated:       timeCreated,
		Skill:             skill,
		Region:            region,
	}
}

type ExitMatchmakingMessage struct {
	TypeDiscriminator string        `json:"$type"`
	UserId            uuidstring.ID `json:"user_id"`
	UserSkill         int           `json:"user_skill"`
}

//func (m *ExitMatchmakingMessage) GetMetaData() map[string]interface{} {
//	return m.MetaData
//}
//
//func (m *ExitMatchmakingMessage) SetMetaData(md map[string]interface{}) {
//	m.MetaData = md
//}
//
//func (m *ExitMatchmakingMessage) Ack(ctx context.Context) error {
//	return m.ack(ctx)
//}
//
//func (m *ExitMatchmakingMessage) SetAck(f func(context.Context) error) {
//	m.ack = f
//}

//func (m *ExitMatchmakingMessage) GetID() string {
//	return m.ID
//}
//
//func (m *ExitMatchmakingMessage) SetID(s string) {
//	m.ID = s
//}

func (m *ExitMatchmakingMessage) GetDiscriminator() string {
	return m.TypeDiscriminator
}

func NewExitMatchmakingMessage(userId uuidstring.ID) ExitMatchmakingMessage {
	return ExitMatchmakingMessage{
		TypeDiscriminator: string(ExitMatchmaking),
		UserId:            userId,
	}
}
