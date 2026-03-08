package storage_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arvarik/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/arvind/whoop-stats/internal/storage"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(ctx context.Context, t *testing.T) (*pgxpool.Pool, func()) {
	pgContainer, err := postgres.Run(ctx,
		"timescale/timescaledb:latest-pg15",
		postgres.WithDatabase("whoop_stats"),
		postgres.WithUsername("whoop_user"),
		postgres.WithPassword("secretpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Run migrations
	runMigrations(t, pool)

	cleanup := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate pgContainer: %s", err)
		}
	}

	return pool, cleanup
}

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	// Simple bare-bones exec of migration files for testing purposes to avoid golang-migrate dependency graph weight
	upSQL1, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000001_init_schema.up.sql"))
	require.NoError(t, err)
	upSQL2, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000002_add_missing_fields.up.sql"))
	require.NoError(t, err)

	ctx := context.Background()

	// Split migration files by semicolon because pgx executes multiple statements in an implicit transaction, 
	// and TimescaleDB's CREATE MATERIALIZED VIEW WITH DATA cannot run inside a transaction block.
	executeSQLStmts(t, ctx, pool, string(upSQL1))
	executeSQLStmts(t, ctx, pool, string(upSQL2))
}

func executeSQLStmts(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string) {
	// A naive split on semicolon. This assumes no semicolons inside string literals.
	// This is sufficient for our specific simple migration files.
	stmts := strings.Split(sql, ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err := pool.Exec(ctx, stmt)
		require.NoError(t, err, "Failed to execute statement: %s", stmt)
	}
}

func TestStorage_UpsertCycleIdempotency(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(ctx, t)
	defer cleanup()

	queries := db.New(pool)
	store := storage.NewStorage(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 1. Create a dummy user
	user, err := queries.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           "123",
		EncryptedAccessToken:  []byte("test"),
		EncryptedRefreshToken: []byte("test"),
	})
	require.NoError(t, err)

	// 2. Mock cycle
	now := time.Now()
	cycle := whoop.Cycle{
		ID:             1,
		Start:          now,
		TimezoneOffset: "-0500",
		Score: &whoop.Score{
			Strain:           14.5,
			Kilojoule:        2000,
			AverageHeartRate: 60,
			MaxHeartRate:     180,
		},
	}

	// 3. Upsert once
	err = store.UpsertCycle(ctx, logger, user.ID, &cycle)
	require.NoError(t, err)

	// 4. Upsert again with modified data to test idempotency/upsert
	cycle.Score.Strain = 15.5
	cycle.Score.Kilojoule = 2100
	err = store.UpsertCycle(ctx, logger, user.ID, &cycle)
	require.NoError(t, err)

	// 5. Verify the DB
	var tNow pgtype.Timestamptz
	err = tNow.Scan(now)
	require.NoError(t, err)
	
	// Wait, the GetCycles query gets items `< start_time`.
	// We'll write a manual query just to grab this one row.
	row := pool.QueryRow(ctx, "SELECT strain, kilojoule FROM cycles WHERE id = $1", cycle.ID)
	var strain, kj float32
	err = row.Scan(&strain, &kj)
	require.NoError(t, err)

	assert.Equal(t, float32(15.5), strain)
	assert.Equal(t, float32(2100), kj)
}
