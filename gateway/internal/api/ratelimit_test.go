package api

import (
	"fmt"
	"testing"
	"time"
)

func TestRateLimiterBoundsDistinctPrincipals(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute)
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	for index := range 10000 {
		if !limiter.Allow(fmt.Sprintf("principal-%d", index)) {
			t.Fatalf("principal %d was rejected before capacity", index)
		}
	}
	if limiter.Allow("over-capacity") {
		t.Fatal("new principal was accepted above bounded capacity")
	}
	now = now.Add(time.Minute)
	if !limiter.Allow("after-expiry") {
		t.Fatal("expired principals were not cleaned")
	}
}
