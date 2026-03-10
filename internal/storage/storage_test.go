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

	ctx := context.Background()

	// Split migration files by semicolon because pgx executes multiple statements in an implicit transaction, 
	// and TimescaleDB's CREATE MATERIALIZED VIEW WITH DATA cannot run inside a transaction block.
	executeSQLStmts(t, ctx, pool, string(upSQL1))
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
	now := time.Now().Truncate(time.Millisecond)
	cycle := whoop.Cycle{
		ID:             1,
		Start:          now,
		TimezoneOffset: "-0500",
		ScoreState:     "SCORED",
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
	cycle.ScoreState = "UPDATED"
	err = store.UpsertCycle(ctx, logger, user.ID, &cycle)
	require.NoError(t, err)

	// 5. Verify the DB
	row := pool.QueryRow(ctx, "SELECT strain, kilojoule, score_state FROM cycles WHERE id = $1", cycle.ID)
	var strain, kj float32
	var scoreState string
	err = row.Scan(&strain, &kj, &scoreState)
	require.NoError(t, err)

	assert.Equal(t, float32(15.5), strain)
	assert.Equal(t, float32(2100), kj)
	assert.Equal(t, "UPDATED", scoreState)
}

func TestStorage_UpsertSleep(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(ctx, t)
	defer cleanup()

	queries := db.New(pool)
	store := storage.NewStorage(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	user, _ := queries.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           "123",
		EncryptedAccessToken:  []byte("test"),
		EncryptedRefreshToken: []byte("test"),
	})

	sleep := whoop.Sleep{
		ID:             "100",
		CycleID:        1,
		Start:          time.Now().Add(-8 * time.Hour),
		End:            time.Now().Add(-1 * time.Hour),
		TimezoneOffset: "+0000",
		ScoreState:     "SCORED",
		Score: &whoop.SleepScore{
			SleepPerformancePercentage: 85,
			SleepNeeded: &whoop.SleepNeeded{
				BaselineMilli: 28800000,
			},
		},
	}

	err := store.UpsertSleep(ctx, logger, user.ID, &sleep)
	require.NoError(t, err)

	var scoreState string
	var baseline int32
	var cycleID int64
	err = pool.QueryRow(ctx, "SELECT score_state, baseline_milli, cycle_id FROM sleeps WHERE id = 100").Scan(&scoreState, &baseline, &cycleID)
	require.NoError(t, err)

	assert.Equal(t, "SCORED", scoreState)
	assert.Equal(t, int32(28800000), baseline)
	assert.Equal(t, int64(1), cycleID)
}

func TestStorage_UpsertWorkout(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(ctx, t)
	defer cleanup()

	queries := db.New(pool)
	store := storage.NewStorage(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	user, _ := queries.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           "123",
		EncryptedAccessToken:  []byte("test"),
		EncryptedRefreshToken: []byte("test"),
	})

	workout := whoop.Workout{
		ID:             "200",
		Start:          time.Now().Add(-2 * time.Hour),
		End:            time.Now().Add(-1 * time.Hour),
		TimezoneOffset: "+0000",
		SportName:      "Running",
		ScoreState:     "SCORED",
	}

	err := store.UpsertWorkout(ctx, logger, user.ID, &workout)
	require.NoError(t, err)

	var sportName, scoreState string
	err = pool.QueryRow(ctx, "SELECT sport_name, score_state FROM workouts WHERE id = 200").Scan(&sportName, &scoreState)
	require.NoError(t, err)

	assert.Equal(t, "Running", sportName)
	assert.Equal(t, "SCORED", scoreState)
}

func TestStorage_UpsertUserProfile(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(ctx, t)
	defer cleanup()

	queries := db.New(pool)
	store := storage.NewStorage(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	user, _ := queries.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           "123",
		EncryptedAccessToken:  []byte("test"),
		EncryptedRefreshToken: []byte("test"),
	})

	profile := whoop.BasicProfile{
		UserID:    123,
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	err := store.UpsertUserProfile(ctx, logger, user.ID, &profile)
	require.NoError(t, err)

	var email string
	err = pool.QueryRow(ctx, "SELECT email FROM user_profiles WHERE id = $1", user.ID).Scan(&email)
	require.NoError(t, err)

	assert.Equal(t, "test@example.com", email)
}

func TestStorage_UpsertRecovery(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(ctx, t)
	defer cleanup()

	queries := db.New(pool)
	store := storage.NewStorage(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	user, _ := queries.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           "123",
		EncryptedAccessToken:  []byte("test"),
		EncryptedRefreshToken: []byte("test"),
	})

	recovery := whoop.Recovery{
		CycleID:    1,
		SleepID:    "100",
		CreatedAt:  time.Now(),
		ScoreState: "SCORED",
		Score: &whoop.RecoveryScore{
			RecoveryScore:   85,
			UserCalibrating: false,
		},
	}

	err := store.UpsertRecovery(ctx, logger, user.ID, &recovery)
	require.NoError(t, err)

	var scoreState string
	var sleepID int64
	var userCalibrating bool
	err = pool.QueryRow(ctx, "SELECT score_state, sleep_id, user_calibrating FROM recoveries WHERE id = 1").Scan(&scoreState, &sleepID, &userCalibrating)
	require.NoError(t, err)

	assert.Equal(t, "SCORED", scoreState)
	assert.Equal(t, int64(100), sleepID)
	assert.False(t, userCalibrating)
}
