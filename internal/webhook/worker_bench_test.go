package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/arvind/whoop-stats/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockQueries struct {
	latency time.Duration
}

func (m *mockQueries) UpdateWebhookEventStatus(ctx context.Context, arg db.UpdateWebhookEventStatusParams) error {
	time.Sleep(m.latency)
	return nil
}

func BenchmarkWebhookStatusUpdates_Baseline(b *testing.B) {
	mock := &mockQueries{latency: 2 * time.Millisecond}
	events := make([]db.WebhookEvent, 50)
	for i := range events {
		events[i].ID = pgtype.UUID{Bytes: [16]byte{byte(i)}, Valid: true}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, event := range events {
			_ = mock.UpdateWebhookEventStatus(context.Background(), db.UpdateWebhookEventStatusParams{
				ID:     event.ID,
				Status: "processed",
			})
		}
	}
}

func BenchmarkWebhookStatusUpdates_Optimized(b *testing.B) {
	mock := &mockQueries{latency: 2 * time.Millisecond}
	events := make([]db.WebhookEvent, 50)
	for i := range events {
		events[i].ID = pgtype.UUID{Bytes: [16]byte{byte(i)}, Valid: true}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var processedIDs []pgtype.UUID
		for _, event := range events {
			processedIDs = append(processedIDs, event.ID)
		}

		time.Sleep(mock.latency) // simulated single query latency
	}
}
