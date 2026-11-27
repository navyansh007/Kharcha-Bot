// Package ratelimit limits how often users and chats can make the bot do work.
// The limits are held in memory, so they apply to a single instance and reset
// when it restarts.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const maxKeys = 10_000

// Keyed is a token bucket for each key.
type Keyed struct {
	limit rate.Limit
	burst int
	idle  time.Duration

	mu      sync.Mutex
	entries map[int64]*entry
	swept   time.Time
}

type entry struct {
	limiter *rate.Limiter
	seen    time.Time
}

// NewKeyed allows burst events at once for each key, refilling one every
// interval. A key unused for as long as a full refill takes is forgotten.
func NewKeyed(interval time.Duration, burst int) *Keyed {
	return &Keyed{
		limit:   rate.Every(interval),
		burst:   burst,
		idle:    interval * time.Duration(burst),
		entries: map[int64]*entry{},
	}
}

// Allow reports whether an event for key may happen at now.
func (k *Keyed) Allow(key int64, now time.Time) bool {
	k.mu.Lock()
	defer k.mu.Unlock()

	e, ok := k.entries[key]
	if !ok {
		k.makeRoom(now)
		e = &entry{limiter: rate.NewLimiter(k.limit, k.burst)}
		k.entries[key] = e
	}
	e.seen = now
	return e.limiter.AllowN(now, 1)
}

func (k *Keyed) makeRoom(now time.Time) {
	if now.Sub(k.swept) >= k.idle {
		for key, e := range k.entries {
			if now.Sub(e.seen) >= k.idle {
				delete(k.entries, key)
			}
		}
		k.swept = now
	}
	for key := range k.entries {
		if len(k.entries) < maxKeys {
			return
		}
		delete(k.entries, key)
	}
}

// Request describes an incoming update.
type Request struct {
	UserID  int64
	ChatID  int64
	IsGroup bool
}

// Guard applies every limit on incoming updates.
type Guard struct {
	user    *Keyed
	daily   *Keyed
	group   *Keyed
	notices *Keyed
	global  *rate.Limiter
}

// NewGuard returns a Guard with the production limits.
func NewGuard() *Guard {
	return &Guard{
		user:    NewKeyed(2*time.Second, 10),
		daily:   NewKeyed(24*time.Hour/500, 500),
		group:   NewKeyed(3*time.Second, 20),
		notices: NewKeyed(time.Minute, 1),
		global:  rate.NewLimiter(5, 30),
	}
}

// Allow reports whether the update may be processed. The sender's own limits are
// checked first, so that one noisy sender does not use up the shared ones.
func (g *Guard) Allow(r Request, now time.Time) bool {
	if !g.user.Allow(r.UserID, now) || !g.daily.Allow(r.UserID, now) {
		return false
	}
	if r.IsGroup && !g.group.Allow(r.ChatID, now) {
		return false
	}
	return g.global.AllowN(now, 1)
}

// ShouldNotify reports whether a refused sender should be told, which happens at
// most once a minute.
func (g *Guard) ShouldNotify(userID int64, now time.Time) bool {
	return g.notices.Allow(userID, now)
}
