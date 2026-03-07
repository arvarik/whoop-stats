package webhook

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/arvind/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	secret string
	db     *db.Queries
	logger *slog.Logger
}

func NewHandler(secret string, pool *pgxpool.Pool, logger *slog.Logger) *Handler {
	return &Handler{
		secret: secret,
		db:     db.New(pool),
		logger: logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	event, err := whoop.ParseWebhook(r, h.secret)
	if err != nil {
		h.logger.Error("Failed to parse webhook", "error", err)
		http.Error(w, "Invalid signature or payload", http.StatusUnauthorized)
		return
	}

	payloadBytes, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal webhook event", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Zero-Data-Loss Requirement: Write to DB immediately, process later.
	record, err := h.db.CreateWebhookEvent(ctx, db.CreateWebhookEventParams{
		Payload: payloadBytes,
		Status:  "pending",
	})
	if err != nil {
		h.logger.Error("Failed to persist webhook event", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Received and persisted webhook event",
		slog.String("trace_id", event.TraceID),
		slog.String("type", event.Type),
		slog.String("event_id", event.ID),
		slog.String("db_id", record.ID.String()),
	)

	// Return 200 OK immediately
	w.WriteHeader(http.StatusOK)
}
