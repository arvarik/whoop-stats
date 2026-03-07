package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
	// Parse CLI flags
	modeFlag := flag.String("mode", "poll", "Operating mode: 'poll' or 'webhook'")
	userIDFlag := flag.String("user", "12345", "WHOOP User ID to poll (only used in poll mode)")
	flag.Parse()

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Load Configuration
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 2. Initialize Database Connection Pool
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
	logger.Info("Successfully connected to TimescaleDB")

	queries := db.New(dbPool)
	store := storage.NewStorage(dbPool)
	authManager := auth.NewManager(cfg, queries, logger)

	// Graceful Shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Create a single poller instance. If in poll mode, we start its loops.
	// If in webhook mode, we just pass it to the API handler for ad-hoc manual sync triggers.
	appPoller := poller.NewPoller(cfg, authManager, store, logger, *userIDFlag)

	if *modeFlag == "poll" {
		// Start Poller loops
		wg.Add(1)
		go func() {
			defer wg.Done()
			appPoller.Start(ctx)
		}()
		
		// Still start API Server so the frontend can hit /api/v1/sync and /api/v1/cycles
		apiHandler := api.NewHandler(queries, dbPool, authManager, store, appPoller, logger)
		mux := api.NewServer(cfg, apiHandler, logger)

		srv := &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
			Handler: mux,
		}

		go func() {
			logger.Info("Starting API Server in Poll Mode", "port", cfg.ServerPort)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Server error", "error", err)
			}
		}()

	} else if *modeFlag == "webhook" {
		// Start Webhook Worker
		worker := webhook.NewWorker(authManager, store, queries, logger)
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Start(ctx)
		}()

		// Start HTTP Server for Webhooks & API
		handler := webhook.NewHandler(cfg.WhoopWebhookSecret, dbPool, logger)

		apiHandler := api.NewHandler(queries, dbPool, authManager, store, appPoller, logger)
		mux := api.NewServer(cfg, apiHandler, logger)
		mux.Handle("/webhook", handler)

		srv := &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
			Handler: mux,
		}

		go func() {
			logger.Info("Starting Webhook & API Server", "port", cfg.ServerPort)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Server error", "error", err)
			}
		}()

		// Handle shutdown for HTTP server
		go func() {
			<-ctx.Done()
			shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelShutdown()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				logger.Error("HTTP server shutdown error", "error", err)
			}
		}()
	} else {
		logger.Error("Unknown mode", "mode", *modeFlag)
		os.Exit(1)
	}

	// Wait for termination signal
	<-quit
	logger.Info("Shutting down gracefully...")

	// Cancel context to stop background workers
	cancel()

	// Wait for workers to finish
	wg.Wait()
	logger.Info("Shutdown complete.")
}
