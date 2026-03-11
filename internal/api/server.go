package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/arvind/whoop-stats/internal/config"
	"github.com/arvind/whoop-stats/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"golang.org/x/time/rate"

	_ "github.com/arvind/whoop-stats/docs" // This will be generated
)

// @title WHOOP Stats API
// @version 1.0
// @description High-performance RESTful API for WHOOP data.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewServer(cfg *config.Config, handler *Handler, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Standard middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.Logger(logger))

	// CORS locked to frontend origin (or wildcard for dev)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// IP-based Rate Limiter (e.g., 20 requests/second, burst of 50)
	r.Use(middleware.IPRateLimiter(rate.Limit(20), 50))

	// Health Check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := handler.pool.Ping(ctx); err != nil {
			http.Error(w, "Database unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler())

	// Protected API Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth([]byte(cfg.EncryptionKey))) // Using EncryptionKey as JWT secret for simplicity

		r.Get("/user/profile", handler.GetProfile)
		r.Get("/cycles", handler.GetCycles)
		r.Get("/sleeps", handler.GetSleeps)
		r.Get("/workouts", handler.GetWorkouts)
		r.Get("/recoveries", handler.GetRecoveries)
		r.Get("/insights", handler.GetInsights)

		r.Post("/sync", handler.PostSync)
	})

	return r
}
