package services

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type LimiterStore struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	rate    rate.Limit
	burst   int
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewLimiterStore(r rate.Limit, burst int) *LimiterStore {
	return &LimiterStore{
		clients: map[string]*clientLimiter{},
		rate:    r,
		burst:   burst,
	}
}

func (s *LimiterStore) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cl, exists := s.clients[key]
	if !exists {
		cl = &clientLimiter{limiter: rate.NewLimiter(s.rate, s.burst), lastSeen: now}
		s.clients[key] = cl
	}
	cl.lastSeen = now

	for k, v := range s.clients {
		if now.Sub(v.lastSeen) > 10*time.Minute {
			delete(s.clients, k)
		}
	}

	return cl.limiter.Allow()
}
