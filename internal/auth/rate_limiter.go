package auth

import (
	"sync"
	"time"
)

type limiterEntry struct {
	WindowStart time.Time
	Count       int
}

type FixedWindowRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	entries map[string]limiterEntry
}

func NewFixedWindowRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		window:  window,
		limit:   limit,
		entries: make(map[string]limiterEntry),
	}
}

func (l *FixedWindowRateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.WindowStart) >= l.window {
		l.entries[key] = limiterEntry{WindowStart: now, Count: 1}
		return true
	}

	if entry.Count >= l.limit {
		return false
	}

	entry.Count++
	l.entries[key] = entry
	return true
}
