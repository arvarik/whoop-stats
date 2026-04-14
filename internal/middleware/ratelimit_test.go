package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestIPRateLimiter_AllowsWithinLimit(t *testing.T) {
	// Allow 10 requests/second with burst of 10
	limiter := IPRateLimiter(rate.Limit(10), 10)

	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should pass
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for first request, got %d", rr.Code)
	}
}

func TestIPRateLimiter_BlocksWhenExceeded(t *testing.T) {
	// Allow 1 request/second with burst of 1
	limiter := IPRateLimiter(rate.Limit(1), 1)

	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request consumes the burst
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for first request, got %d", rr.Code)
	}

	// Second request should be rate limited
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for second request, got %d", rr.Code)
	}
}

func TestIPRateLimiter_IndependentPerIP(t *testing.T) {
	// Allow 1 request/second with burst of 1
	limiter := IPRateLimiter(rate.Limit(1), 1)

	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP 1 uses its burst
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200 for IP1 first request, got %d", rr1.Code)
	}

	// IP 2 should still be allowed (independent limiter)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.2:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200 for IP2 first request (different IP), got %d", rr2.Code)
	}

	// IP 1 should now be limited
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req1)

	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for IP1 second request, got %d", rr3.Code)
	}
}
