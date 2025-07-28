package matchmake

import (
	"time"
)

const (
	BaseSkillThreshold  = 20
	SkillThresholdDelta = 10
	thirtySecInterval   = time.Second * 30
)

func calculateSkillThreshold(t time.Time) int {
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

func calculateMinMaxSkill(mySkill int, roomCreatedAt time.Time) (int, int) {
	skillDiffAllowed := calculateSkillThreshold(roomCreatedAt)
	minSkill := max(0, mySkill-skillDiffAllowed)
	maxSkill := mySkill + skillDiffAllowed
	return minSkill, maxSkill
}
