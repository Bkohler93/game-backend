package auth

import (
	"testing"
)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter()

	ip := "1234"

	t.Run("first request is always allowed", func(t *testing.T) {
		if rl.DenyGuestRequest(ip) {
			t.Errorf("should not have denied initial request")
		}
	})

	t.Run("second request sent immediately after should be denied", func(t *testing.T) {
		if !rl.DenyGuestRequest(ip) {
			t.Errorf("should have denied the second request made immediately after")
		}
	})

	// this test runs for an extended amount of time
	//t.Run("make a request 6 seconds since last request should work", func(t *testing.T) {
	//	time.Sleep(time.Second * 6)
	//	if rl.DenyGuestRequest(ip) {
	//		t.Errorf("should not have denied request made 6 seconds since last one")
	//	}
	//})
}
