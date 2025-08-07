package room

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
)

const (
	BaseSkillThreshold  = 20
	SkillThresholdDelta = 10
	thirtySecInterval   = time.Second * 30
)

var SortedByCreatedAtFunc = func(a, b Room) int {
	if time.Unix(a.CreatedAt, 0).Before(time.Unix(b.CreatedAt, 0)) {
		return -1
	}
	return 1
}

func RandomRoom() Room {
	id := uuidstring.NewID()
	return Room{
		RoomId:       id,
		PlayerCount:  1,
		AverageSkill: 100,
		Region:       "na",
		PlayerIds:    []uuidstring.ID{id},
		CreatedAt:    time.Now().Unix(),
		IsFull:       0,
	}
}

func CalculateSkillThreshold(tUnix int64) int {
	skillDiffAllowed := BaseSkillThreshold
	t := time.Unix(tUnix, 0)
	for i := 1; i > 0; i++ {
		if time.Since(t) > thirtySecInterval*time.Duration(i) {
			skillDiffAllowed += SkillThresholdDelta * i
		} else {
			break
		}
	}
	return skillDiffAllowed
}

func CalculateMinMaxSkill(mySkill int, roomCreatedAt int64) (int, int) {
	skillDiffAllowed := CalculateSkillThreshold(roomCreatedAt)
	minSkill := max(0, mySkill-skillDiffAllowed)
	maxSkill := mySkill + skillDiffAllowed
	return minSkill, maxSkill
}
