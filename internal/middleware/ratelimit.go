package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
}

type visitor struct {
	lastSeen time.Time
	tokens   int
}

func NewRateLimiter(requestsPerMinute int) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
	}

	// Cleanup old visitors every minute
	go rl.cleanup()

	return rl
}

func (rl *rateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			rl.visitors[ip] = &visitor{
				lastSeen: time.Now(),
				tokens:   10, // 10 requests per minute
			}
			v = rl.visitors[ip]
		}

		// Refill tokens based on time passed
		elapsed := time.Since(v.lastSeen)
		v.tokens += int(elapsed.Minutes() * 10)
		if v.tokens > 10 {
			v.tokens = 10
		}
		v.lastSeen = time.Now()

		if v.tokens > 0 {
			v.tokens--
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
		} else {
			rl.mu.Unlock()
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		}
	})
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
