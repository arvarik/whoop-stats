// Package poller implements periodic data fetching from the WHOOP API.
// It runs independent polling loops for cycles, workouts, sleeps, and user
// profile data, with configurable intervals and built-in rate limiting.
package poller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/arvarik/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/auth"
	"github.com/arvind/whoop-stats/internal/config"
	"github.com/arvind/whoop-stats/internal/storage"
	"golang.org/x/time/rate"
)

// Poller periodically fetches WHOOP data and upserts it into the database.
// It manages separate polling loops for different data types and enforces
// API rate limits across all concurrent requests.
type Poller struct {
	cfg         *config.Config
	authManager *auth.Manager
	storage     *storage.Storage
	logger      *slog.Logger
	whoopUserID string

	// limiter enforces WHOOP API rate limits across all polling goroutines.
	limiter *rate.Limiter

	lastOffpeakSleepPoll time.Time
}

// NewPoller creates a new Poller. The rate limiter allows 2 requests per second
// to stay well under the WHOOP API rate limit.
func NewPoller(cfg *config.Config, authManager *auth.Manager, store *storage.Storage, logger *slog.Logger, whoopUserID string) *Poller {
	return &Poller{
		cfg:         cfg,
		authManager: authManager,
		storage:     store,
		logger:      logger,
		whoopUserID: whoopUserID,
		limiter:     rate.NewLimiter(rate.Every(500*time.Millisecond), 2),
	}
}

// Start begins all polling loops and blocks until the context is cancelled.
func (p *Poller) Start(ctx context.Context) {
	p.logger.Info("Starting polling engine", "user_id", p.whoopUserID)

	type loopConfig struct {
		name     string
		envKey   string
		pollFunc func(context.Context) error
	}

	loops := []loopConfig{
		{"cycles_recoveries", p.cfg.PollIntervalCycle, p.pollCyclesAndRecoveries},
		{"workouts", p.cfg.PollIntervalWorkout, p.pollWorkouts},
		{"sleeps", p.cfg.PollIntervalSleep, p.pollSleeps},
		{"profile", p.cfg.PollIntervalProfile, p.pollUserProfile},
	}

	var wg sync.WaitGroup
	for _, lc := range loops {
		interval, err := time.ParseDuration(lc.envKey)
		if err != nil {
			p.logger.Error("Invalid poll interval, using 1h default", "task", lc.name, "value", lc.envKey, "error", err)
			interval = 1 * time.Hour
		}

		wg.Add(1)
		go func(name string, interval time.Duration, fn func(context.Context) error) {
			defer wg.Done()
			p.pollLoop(ctx, name, interval, fn)
		}(lc.name, interval, lc.pollFunc)
	}

	wg.Wait()
}

// waitForRateLimit blocks until the rate limiter allows a request, logging
// if the wait exceeds 600ms (indicating throttling pressure).
func (p *Poller) waitForRateLimit(ctx context.Context, task string) error {
	start := time.Now()
	if err := p.limiter.Wait(ctx); err != nil {
		return err
	}
	if d := time.Since(start); d > 600*time.Millisecond {
		p.logger.Debug("Rate limiter throttled request", "task", task, "wait", d)
	}
	return nil
}

// pollLoop runs a polling function on a fixed interval with initial jitter
// to prevent thundering herd on startup.
func (p *Poller) pollLoop(ctx context.Context, name string, interval time.Duration, pollFunc func(ctx context.Context) error) {
	// Initial jitter (0-60s) to stagger startup across loops
	jitter := time.Duration(rand.Intn(60)) * time.Second
	select {
	case <-time.After(jitter):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		start := time.Now()
		p.logger.Info("Starting poll run", "task", name)

		if err := pollFunc(ctx); err != nil {
			p.logger.Error("Poll run failed", "task", name, "error", err, "duration", time.Since(start))
		} else {
			p.logger.Info("Poll run succeeded", "task", name, "duration", time.Since(start))
		}

		select {
		case <-ctx.Done():
			p.logger.Info("Stopping poll loop", "task", name)
			return
		case <-ticker.C:
		}
	}
}

// RunAdHocSync performs a one-off sync of all data types. Used by the /sync endpoint.
func (p *Poller) RunAdHocSync(ctx context.Context, whoopUserID string) {
	p.logger.Info("Starting ad-hoc sync", "user_id", whoopUserID)

	for _, step := range []struct {
		name string
		fn   func(context.Context) error
	}{
		{"cycles_recoveries", p.pollCyclesAndRecoveries},
		{"workouts", p.pollWorkouts},
		{"sleeps", p.pollSleeps},
		{"profile", p.pollUserProfile},
	} {
		if err := step.fn(ctx); err != nil {
			p.logger.Error("Ad-hoc sync step failed", "step", step.name, "error", err)
		}
	}

	p.logger.Info("Ad-hoc sync completed", "user_id", whoopUserID)
}

func (p *Poller) pollCyclesAndRecoveries(ctx context.Context) error {
	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting client: %w", err)
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting internal user ID: %w", err)
	}

	// --- Cycles ---
	if err := p.waitForRateLimit(ctx, "cycles"); err != nil {
		return err
	}
	cyclesPage, err := client.Cycle.List(ctx, nil)
	totalCycles := 0
	for page := 1; ; page++ {
		if err != nil {
			return fmt.Errorf("listing cycles (page %d): %w", page, err)
		}
		totalCycles += len(cyclesPage.Records)
		p.logger.Debug("Processing cycles page", "page", page, "records", len(cyclesPage.Records))

		for _, cycle := range cyclesPage.Records {
			if err := p.storage.UpsertCycle(ctx, internalUserID, &cycle); err != nil {
				p.logger.Error("Failed to upsert cycle", "id", cycle.ID, "error", err)
			}
		}

		cyclesPage, err = cyclesPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitForRateLimit(ctx, "cycles_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Cycles sync completed", "total", totalCycles)

	// --- Recoveries ---
	if err := p.waitForRateLimit(ctx, "recoveries"); err != nil {
		return err
	}
	recoveriesPage, err := client.Recovery.List(ctx, nil)
	totalRecoveries := 0
	for page := 1; ; page++ {
		if err != nil {
			return fmt.Errorf("listing recoveries (page %d): %w", page, err)
		}
		totalRecoveries += len(recoveriesPage.Records)
		p.logger.Debug("Processing recoveries page", "page", page, "records", len(recoveriesPage.Records))

		for _, recovery := range recoveriesPage.Records {
			if err := p.storage.UpsertRecovery(ctx, internalUserID, &recovery); err != nil {
				p.logger.Error("Failed to upsert recovery", "cycle_id", recovery.CycleID, "error", err)
			}
		}

		recoveriesPage, err = recoveriesPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitForRateLimit(ctx, "recoveries_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Recoveries sync completed", "total", totalRecoveries)

	return nil
}

func (p *Poller) pollWorkouts(ctx context.Context) error {
	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting client: %w", err)
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting internal user ID: %w", err)
	}

	if err := p.waitForRateLimit(ctx, "workouts"); err != nil {
		return err
	}
	workoutsPage, err := client.Workout.List(ctx, nil)
	totalWorkouts := 0
	for page := 1; ; page++ {
		if err != nil {
			return fmt.Errorf("listing workouts (page %d): %w", page, err)
		}
		totalWorkouts += len(workoutsPage.Records)
		p.logger.Debug("Processing workouts page", "page", page, "records", len(workoutsPage.Records))

		for _, w := range workoutsPage.Records {
			if err := p.storage.UpsertWorkout(ctx, internalUserID, &w); err != nil {
				p.logger.Error("Failed to upsert workout", "id", w.ID, "error", err)
			}
		}

		workoutsPage, err = workoutsPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitForRateLimit(ctx, "workouts_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Workouts sync completed", "total", totalWorkouts)
	return nil
}

func (p *Poller) pollSleeps(ctx context.Context) error {
	// Adaptive polling: more frequent during peak sleep-data hours (6 AM – 12 PM)
	hour := time.Now().Hour()
	isPeak := hour >= 6 && hour <= 12
	if !isPeak {
		offpeakInterval, err := time.ParseDuration(p.cfg.PollIntervalSleepOffpeak)
		if err != nil {
			offpeakInterval = 4 * time.Hour
		}
		if time.Since(p.lastOffpeakSleepPoll) < offpeakInterval {
			p.logger.Debug("Skipping sleep poll (off-peak, interval not reached)", "last_run", p.lastOffpeakSleepPoll)
			return nil
		}
		p.lastOffpeakSleepPoll = time.Now()
	}

	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting client: %w", err)
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting internal user ID: %w", err)
	}

	if err := p.waitForRateLimit(ctx, "sleeps"); err != nil {
		return err
	}
	sleepsPage, err := client.Sleep.List(ctx, nil)
	totalSleeps := 0
	for page := 1; ; page++ {
		if err != nil {
			return fmt.Errorf("listing sleeps (page %d): %w", page, err)
		}
		totalSleeps += len(sleepsPage.Records)
		p.logger.Debug("Processing sleeps page", "page", page, "records", len(sleepsPage.Records))

		for _, s := range sleepsPage.Records {
			if err := p.storage.UpsertSleep(ctx, internalUserID, &s); err != nil {
				p.logger.Error("Failed to upsert sleep", "id", s.ID, "error", err)
			}
		}

		sleepsPage, err = sleepsPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitForRateLimit(ctx, "sleeps_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Sleeps sync completed", "total", totalSleeps)
	return nil
}

func (p *Poller) pollUserProfile(ctx context.Context) error {
	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting client: %w", err)
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return fmt.Errorf("getting internal user ID: %w", err)
	}

	if err := p.waitForRateLimit(ctx, "profile"); err != nil {
		return err
	}

	profile, err := client.User.GetBasicProfile(ctx)
	if err != nil {
		return fmt.Errorf("fetching profile: %w", err)
	}
	if err := p.storage.UpsertUserProfile(ctx, internalUserID, profile); err != nil {
		return fmt.Errorf("upserting profile: %w", err)
	}

	if err := p.waitForRateLimit(ctx, "measurement"); err != nil {
		return err
	}
	measurement, err := client.User.GetBodyMeasurement(ctx)
	if err != nil {
		return fmt.Errorf("fetching body measurement: %w", err)
	}
	if err := p.storage.UpsertBodyMeasurement(ctx, internalUserID, measurement); err != nil {
		return fmt.Errorf("upserting body measurement: %w", err)
	}

	p.logger.Info("Profile and measurements updated")
	return nil
}
