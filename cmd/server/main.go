// Package main is the entry point for the whoop-stats server.
// It supports two operating modes:
//   - poll: Periodically fetches data from the WHOOP API (for homelabs behind NAT)
//   - webhook: Receives push notifications from WHOOP (requires public endpoint)
//
// In both modes, a REST API server is started to serve the frontend dashboard.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/arvind/whoop-stats/internal/api"
	"github.com/arvind/whoop-stats/internal/auth"
	"github.com/arvind/whoop-stats/internal/config"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/arvind/whoop-stats/internal/poller"
	"github.com/arvind/whoop-stats/internal/storage"
	"github.com/arvind/whoop-stats/internal/webhook"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	modeFlag := flag.String("mode", "poll", "Operating mode: 'poll' or 'webhook'")
	userIDFlag := flag.String("user", "12345", "WHOOP User ID to poll (only used in poll mode)")
	flag.Parse()

	// Validate mode early
	if *modeFlag != "poll" && *modeFlag != "webhook" {
		fmt.Fprintf(os.Stderr, "Error: unknown mode %q (must be 'poll' or 'webhook')\n", *modeFlag)
		os.Exit(1)
	}

	// Bootstrap logger for startup messages
	setupLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Load and validate configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		setupLogger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Re-initialize logger with configured level
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	logger.Info("Starting whoop-stats",
		"mode", *modeFlag,
		"port", cfg.ServerPort,
		"log_level", cfg.LogLevel,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to parse database URL", "error", err)
		os.Exit(1)
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("Failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		logger.Error("Database is not reachable", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to TimescaleDB")

	queries := db.New(dbPool)
	store := storage.NewStorage(dbPool, logger)
	authManager := auth.NewManager(cfg, queries, logger)

	// Graceful shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Poller is created in both modes — in webhook mode it's only used for ad-hoc /sync triggers
	appPoller := poller.NewPoller(cfg, authManager, store, logger, *userIDFlag)

	if *modeFlag == "poll" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			appPoller.Start(ctx)
		}()
	}

	// Build the API router
	apiHandler := api.NewHandler(queries, dbPool, authManager, store, appPoller, logger)
	mux := api.NewServer(cfg, apiHandler, logger)

	// In webhook mode, attach the webhook endpoint
	if *modeFlag == "webhook" {
		worker := webhook.NewWorker(authManager, store, queries, logger)
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Start(ctx)
		}()

		handler := webhook.NewHandler(cfg.WhoopWebhookSecret, dbPool, logger)
		mux.Handle("/webhook", handler)
	}

	// Start HTTP server (shared for both modes)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("API server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	<-quit
	logger.Info("Shutting down gracefully...")

	// Cancel context to stop background workers (poller, webhook worker)
	cancel()

	// Shut down HTTP server with a deadline
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	// Wait for background workers to finish
	wg.Wait()
	logger.Info("Shutdown complete")
}

// parseLogLevel maps a string log level to the corresponding slog.Level.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
