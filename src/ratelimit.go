package src

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type Ratelimit struct {
	max_requests int
	window_secs  int
	window_start time.Time
	requests     map[string]int
	mu           sync.Mutex
}

func RatelimitNew(max_requests int, window_secs int) Ratelimit {
	return Ratelimit{
		max_requests: max_requests,
		window_secs:  window_secs,
		window_start: time.Now(),
		requests:     make(map[string]int),
	}
}

func (rl *Ratelimit) Allow(r *http.Request) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	ip := r.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	now := time.Now()

	if now.Sub(rl.window_start) > time.Duration(rl.window_secs)*time.Second {
		rl.window_start = now
		rl.requests = make(map[string]int)
	}

	rl.requests[ip]++

	return rl.requests[ip] <= rl.max_requests
}
