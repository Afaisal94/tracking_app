package middleware

import (
	"net/http"
	"sync"
	"time"

	"tracking-backend/internal/utils"
)

type visitor struct {
	tokens float64
	last   time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      float64
	burst    int
}

func NewRateLimiter(rps int, burst int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		rps:      float64(rps),
		burst:    burst,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, ok := rl.visitors[key]
	if !ok {
		rl.visitors[key] = &visitor{tokens: float64(rl.burst - 1), last: now}
		return true
	}
	elapsed := now.Sub(v.last).Seconds()
	v.tokens += elapsed * rl.rps
	if v.tokens > float64(rl.burst) {
		v.tokens = float64(rl.burst)
	}
	v.last = now
	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		if !rl.Allow(ip) {
			utils.Error(w, http.StatusTooManyRequests, "too many requests, please try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Cleanup periodically removes stale visitors
func (rl *RateLimiter) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			for k, v := range rl.visitors {
				if time.Since(v.last) > 5*time.Minute {
					delete(rl.visitors, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
}
