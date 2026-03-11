// Package webhook implements the WHOOP webhook ingestion pipeline consisting of:
//   - Handler: receives webhook events and stores them immediately (inbox pattern)
//   - Worker: background processor that reads pending events and syncs full objects
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/arvarik/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/auth"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/arvind/whoop-stats/internal/storage"
	"golang.org/x/time/rate"
)

// Worker processes pending webhook events from the database. It runs on a
// 5-second polling interval and fetches up to 50 events per batch.
type Worker struct {
	authManager *auth.Manager
	storage     *storage.Storage
	logger      *slog.Logger
	db          *db.Queries
	limiter     *rate.Limiter
}

// NewWorker creates a new webhook background worker.
func NewWorker(authManager *auth.Manager, store *storage.Storage, queries *db.Queries, logger *slog.Logger) *Worker {
	return &Worker{
		authManager: authManager,
		storage:     store,
		db:          queries,
		logger:      logger,
		limiter:     rate.NewLimiter(rate.Every(500*time.Millisecond), 2),
	}
}

// Start begins the webhook processing loop, blocking until the context is cancelled.
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Starting webhook background worker")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping webhook background worker")
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

func (w *Worker) processPendingEvents(ctx context.Context) {
	events, err := w.db.GetPendingWebhookEvents(ctx, 50)
	if err != nil {
		w.logger.Error("Failed to fetch pending webhook events", "error", err)
		return
	}

	for _, record := range events {
		if err := w.processEvent(ctx, record); err != nil {
			w.logger.Error("Failed to process webhook event", "db_id", record.ID.String(), "error", err)
			// Mark as failed to prevent retry loops. In production, consider
			// adding a retry_count column with exponential backoff.
			_ = w.db.UpdateWebhookEventStatus(ctx, db.UpdateWebhookEventStatusParams{
				ID:     record.ID,
				Status: "failed",
			})
		} else {
			_ = w.db.UpdateWebhookEventStatus(ctx, db.UpdateWebhookEventStatusParams{
				ID:     record.ID,
				Status: "processed",
			})
			w.logger.Info("Processed webhook event", "db_id", record.ID.String())
		}
	}
}

func (w *Worker) processEvent(ctx context.Context, record db.WebhookEvent) error {
	var event whoop.WebhookEvent
	if err := json.Unmarshal(record.Payload, &event); err != nil {
		return fmt.Errorf("invalid payload JSON: %w", err)
	}

	whoopUserIDStr := strconv.Itoa(event.UserID)
	traceLogger := w.logger.With("trace_id", event.TraceID, "event_type", event.Type, "whoop_user_id", event.UserID)

	client, err := w.authManager.GetClient(ctx, whoopUserIDStr)
	if err != nil {
		return fmt.Errorf("getting client for user %s: %w", whoopUserIDStr, err)
	}

	internalUserID, err := w.authManager.GetInternalUserID(ctx, whoopUserIDStr)
	if err != nil {
		return fmt.Errorf("getting internal user ID: %w", err)
	}

	if err := w.limiter.Wait(ctx); err != nil {
		return err
	}

	traceLogger.Debug("Processing webhook event")

	// Fetch full object from WHOOP API and upsert into database
	switch event.Type {
	case "recovery.updated":
		objectID, err := strconv.Atoi(event.ID)
		if err != nil {
			return fmt.Errorf("invalid recovery ID %q: %w", event.ID, err)
		}
		traceLogger.Info("Fetching updated recovery", "cycle_id", objectID)
		obj, err := client.Recovery.GetByID(ctx, objectID)
		if err != nil {
			return fmt.Errorf("fetching recovery: %w", err)
		}
		return w.storage.UpsertRecovery(ctx, internalUserID, obj)

	case "cycle.updated":
		objectID, err := strconv.Atoi(event.ID)
		if err != nil {
			return fmt.Errorf("invalid cycle ID %q: %w", event.ID, err)
		}
		traceLogger.Info("Fetching updated cycle", "cycle_id", objectID)
		obj, err := client.Cycle.GetByID(ctx, objectID)
		if err != nil {
			return fmt.Errorf("fetching cycle: %w", err)
		}
		return w.storage.UpsertCycle(ctx, internalUserID, obj)

	case "workout.updated":
		traceLogger.Info("Fetching updated workout", "workout_id", event.ID)
		obj, err := client.Workout.GetByID(ctx, event.ID)
		if err != nil {
			return fmt.Errorf("fetching workout: %w", err)
		}
		return w.storage.UpsertWorkout(ctx, internalUserID, obj)

	case "sleep.updated":
		traceLogger.Info("Fetching updated sleep", "sleep_id", event.ID)
		obj, err := client.Sleep.GetByID(ctx, event.ID)
		if err != nil {
			return fmt.Errorf("fetching sleep: %w", err)
		}
		return w.storage.UpsertSleep(ctx, internalUserID, obj)

	default:
		traceLogger.Warn("Unknown webhook event type, marking as processed")
		return nil
	}
}
