package room

import (
	"time"

	"github.com/bkohler93/game-backend/pkg/stringuuid"
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
	id := stringuuid.NewStringUUID()
	return Room{
		RoomId:       id,
		PlayerCount:  1,
		AverageSkill: 100,
		Region:       "na",
		PlayerIds:    []stringuuid.StringUUID{id},
		CreatedAt:    time.Now().Unix(),
		IsFull:       0,
	}
}

func CalculateSkillThreshold(t time.Time) int {
	skillDiffAllowed := BaseSkillThreshold
	for i := 1; i > 0; i++ {
		if time.Since(t) > thirtySecInterval*time.Duration(i) {
			skillDiffAllowed += SkillThresholdDelta * i
		} else {
			break
		}
	}
	return skillDiffAllowed
}

func CalculateMinMaxSkill(mySkill int, roomCreatedAt time.Time) (int, int) {
	skillDiffAllowed := CalculateSkillThreshold(roomCreatedAt)
	minSkill := max(0, mySkill-skillDiffAllowed)
	maxSkill := mySkill + skillDiffAllowed
	return minSkill, maxSkill
}
