package ratelimit

import (
	"testing"
	"time"
)

var start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func allowed(g *Guard, r Request, n int, at time.Time) int {
	count := 0
	for range n {
		if g.Allow(r, at) {
			count++
		}
	}
	return count
}

func TestGuardLimitsOneSenderWithoutBlockingOthers(t *testing.T) {
	g := NewGuard()
	if got := allowed(g, Request{UserID: 1}, 1000, start); got != 10 {
		t.Errorf("burst allowed = %d, want 10", got)
	}
	if !g.Allow(Request{UserID: 2}, start) {
		t.Error("another sender was refused")
	}
	if !g.Allow(Request{UserID: 1}, start.Add(2*time.Second)) {
		t.Error("sender was not allowed again after the refill")
	}
}

func TestGuardDailyLimit(t *testing.T) {
	g := NewGuard()
	count := 0
	for i := range 3000 {
		if g.Allow(Request{UserID: 1}, start.Add(time.Duration(i)*2*time.Second)) {
			count++
		}
	}
	if count < 500 || count > 540 {
		t.Errorf("allowed over 100 minutes = %d, want about 500 plus the refill", count)
	}
}

func TestGuardGroupAndGlobalLimits(t *testing.T) {
	g := NewGuard()
	group := 0
	for user := range int64(100) {
		if g.Allow(Request{UserID: user, ChatID: -1, IsGroup: true}, start) {
			group++
		}
	}
	if group != 20 {
		t.Errorf("group burst = %d, want 20", group)
	}
	private := 0
	for user := range int64(100) {
		if g.Allow(Request{UserID: 1000 + user}, start) {
			private++
		}
	}
	if private != 10 {
		t.Errorf("allowed after the group used 20 of 30 = %d, want 10", private)
	}
}

func TestKeyedStaysBounded(t *testing.T) {
	k := NewKeyed(time.Second, 1)
	for key := range int64(maxKeys + 500) {
		k.Allow(key, start)
	}
	if len(k.entries) > maxKeys {
		t.Errorf("entries = %d, want at most %d", len(k.entries), maxKeys)
	}
}

func TestNotifyOncePerMinute(t *testing.T) {
	g := NewGuard()
	if !g.ShouldNotify(1, start) || g.ShouldNotify(1, start.Add(time.Second)) {
		t.Error("notice was not limited")
	}
	if !g.ShouldNotify(1, start.Add(time.Minute)) {
		t.Error("notice was not allowed after a minute")
	}
}
