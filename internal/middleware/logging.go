// Package middleware provides HTTP middleware for the whoop-stats API server.

package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Logger returns middleware that logs HTTP requests using structured slog output.
// Health check requests (/healthz) are skipped to avoid noisy logs when behind
// a load balancer or orchestrator.
func Logger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging health check probes to reduce noise
			if strings.HasPrefix(r.URL.Path, "/healthz") {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("HTTP request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.Status()),
					slog.Duration("duration", time.Since(start)),
					slog.String("req_id", middleware.GetReqID(r.Context())),
					slog.String("ip", r.RemoteAddr),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
