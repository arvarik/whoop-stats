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

type Worker struct {
	authManager *auth.Manager
	storage     *storage.Storage
	logger      *slog.Logger
	db          *db.Queries
	limiter     *rate.Limiter
}

func NewWorker(authManager *auth.Manager, store *storage.Storage, queries *db.Queries, logger *slog.Logger) *Worker {
	return &Worker{
		authManager: authManager,
		storage:     store,
		db:          queries,
		logger:      logger,
		limiter:     rate.NewLimiter(rate.Every(500*time.Millisecond), 2),
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Starting Webhook Background Worker")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping Webhook Background Worker")
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

func (w *Worker) processPendingEvents(ctx context.Context) {
	// Fetch up to 50 pending events
	events, err := w.db.GetPendingWebhookEvents(ctx, 50)
	if err != nil {
		w.logger.Error("Failed to fetch pending webhook events", "error", err)
		return
	}

	for _, record := range events {
		err := w.processEvent(ctx, record)
		if err != nil {
			w.logger.Error("Failed to process webhook event", "db_id", record.ID.String(), "error", err)
			// We can leave it pending and retry later, or implement a dead-letter queue.
			// For now, mark as failed so we don't get stuck.
			_ = w.db.UpdateWebhookEventStatus(ctx, db.UpdateWebhookEventStatusParams{
				ID:     record.ID,
				Status: "failed",
			})
		} else {
			_ = w.db.UpdateWebhookEventStatus(ctx, db.UpdateWebhookEventStatusParams{
				ID:     record.ID,
				Status: "processed",
			})
			w.logger.Info("Successfully processed webhook event", "db_id", record.ID.String())
		}
	}
}

func (w *Worker) processEvent(ctx context.Context, record db.WebhookEvent) error {
	var event whoop.WebhookEvent
	if err := json.Unmarshal(record.Payload, &event); err != nil {
		return fmt.Errorf("invalid payload json: %w", err)
	}

	whoopUserIDStr := strconv.Itoa(event.UserID)
	traceLogger := w.logger.With("trace_id", event.TraceID, "event_type", event.Type, "whoop_user_id", event.UserID)

	client, err := w.authManager.GetClient(ctx, whoopUserIDStr)
	if err != nil {
		return fmt.Errorf("failed to get client for user %s: %w", whoopUserIDStr, err)
	}

	internalUserID, err := w.authManager.GetInternalUserID(ctx, whoopUserIDStr)
	if err != nil {
		return fmt.Errorf("failed to get internal user id: %w", err)
	}

	if err := w.limiter.Wait(ctx); err != nil {
		return err
	}

	traceLogger.Debug("Processing incoming webhook event")

	// Fetch the full object based on type and upsert
	switch event.Type {
	case "recovery.updated":
		objectID, err := strconv.Atoi(event.ID)
		if err != nil {
			return fmt.Errorf("invalid object id %s: %w", event.ID, err)
		}
		traceLogger.Info("Fetching updated recovery", "cycle_id", objectID)
		obj, err := client.Recovery.GetByID(ctx, objectID)
		if err != nil {
			return err
		}
		return w.storage.UpsertRecovery(ctx, traceLogger, internalUserID, obj)

	case "cycle.updated":
		objectID, err := strconv.Atoi(event.ID)
		if err != nil {
			return fmt.Errorf("invalid object id %s: %w", event.ID, err)
		}
		traceLogger.Info("Fetching updated cycle", "cycle_id", objectID)
		obj, err := client.Cycle.GetByID(ctx, objectID)
		if err != nil {
			return err
		}
		return w.storage.UpsertCycle(ctx, traceLogger, internalUserID, obj)

	case "workout.updated":
		traceLogger.Info("Fetching updated workout", "workout_id", event.ID)
		obj, err := client.Workout.GetByID(ctx, event.ID)
		if err != nil {
			return err
		}
		return w.storage.UpsertWorkout(ctx, traceLogger, internalUserID, obj)

	case "sleep.updated":
		traceLogger.Info("Fetching updated sleep", "sleep_id", event.ID)
		obj, err := client.Sleep.GetByID(ctx, event.ID)
		if err != nil {
			return err
		}
		return w.storage.UpsertSleep(ctx, traceLogger, internalUserID, obj)

	default:
		traceLogger.Warn("Unknown webhook event type received")
		return nil // Return nil so we mark it processed and ignore it
	}
}
