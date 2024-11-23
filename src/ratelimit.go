package src

import (
	"sync"
	"time"
)

type Ratelimit struct {
	max_requests uint32
	window_secs  uint32
	window_start time.Time
	requests     map[string]uint32
	mu           sync.Mutex
}

func RatelimitNew(max_requests uint32, window_secs uint32) Ratelimit {
	return Ratelimit{
		max_requests: max_requests,
		window_secs:  window_secs,
		window_start: time.Now(),
		requests:     make(map[string]uint32),
	}
}

func (rl *Ratelimit) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if now.Sub(rl.window_start) > time.Duration(rl.window_secs)*time.Second {
		rl.window_start = now
		rl.requests = make(map[string]uint32)
	}

	rl.requests[ip]++

	return rl.requests[ip] <= rl.max_requests
}
