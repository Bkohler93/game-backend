package auth

import "time"

const (
	AllowableGuestRequestPeriod = time.Second * 5
)

type RateLimiter struct {
	guestRequests map[string]time.Time
}

func (l *RateLimiter) DenyGuestRequest(ip string) bool {
	lastReqTime, ok := l.guestRequests[ip]
	l.guestRequests[ip] = time.Now()
	if !ok {
		return false
	}

	if time.Since(lastReqTime) < AllowableGuestRequestPeriod {
		return true
	}
	return false
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{make(map[string]time.Time)}
}
