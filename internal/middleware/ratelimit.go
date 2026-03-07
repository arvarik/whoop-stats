package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

func IPRateLimiter(r rate.Limit, b int) func(next http.Handler) http.Handler {
	var mu sync.Mutex
	visitors := make(map[string]*rate.Limiter)

	getVisitor := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		limiter, exists := visitors[ip]
		if !exists {
			limiter = rate.NewLimiter(r, b)
			visitors[ip] = limiter
		}

		return limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Basic IP extraction. In a real app behind a proxy, we'd check X-Forwarded-For or X-Real-IP
			ip := req.RemoteAddr
			limiter := getVisitor(ip)
			if !limiter.Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}
