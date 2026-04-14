package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPRateLimiter returns middleware that enforces per-IP request rate limiting.
// Stale entries are cleaned up every 10 minutes to prevent unbounded memory growth.
func IPRateLimiter(r rate.Limit, b int) func(next http.Handler) http.Handler {
	var mu sync.Mutex
	type visitor struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	visitors := make(map[string]*visitor)

	// Background goroutine cleans up stale entries every 10 minutes
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 30*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ip := req.RemoteAddr

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				v = &visitor{limiter: rate.NewLimiter(r, b)}
				visitors[ip] = v
			}
			v.lastSeen = time.Now()
			mu.Unlock()

			if !v.limiter.Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}
