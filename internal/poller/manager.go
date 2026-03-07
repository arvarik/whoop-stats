package poller

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/arvind/whoop-stats/internal/auth"
	"github.com/arvind/whoop-stats/internal/config"
	"github.com/arvind/whoop-stats/internal/storage"
	"golang.org/x/time/rate"
)

type Poller struct {
	cfg         *config.Config
	authManager *auth.Manager
	storage     *storage.Storage
	logger      *slog.Logger
	whoopUserID string

	// Rate limiter to respect API limits across all polling goroutines
	limiter *rate.Limiter
}

func NewPoller(cfg *config.Config, authManager *auth.Manager, store *storage.Storage, logger *slog.Logger, whoopUserID string) *Poller {
	// WHOOP API limits: typically conservative e.g. 2 requests per second
	limiter := rate.NewLimiter(rate.Every(500*time.Millisecond), 2)

	return &Poller{
		cfg:         cfg,
		authManager: authManager,
		storage:     store,
		logger:      logger,
		whoopUserID: whoopUserID,
		limiter:     limiter,
	}
}

func (p *Poller) Start(ctx context.Context) {
	p.logger.Info("Starting Polling Engine", "user_id", p.whoopUserID)

	// Intervals from config
	cycleInterval, _ := time.ParseDuration(p.cfg.PollIntervalCycle)
	workoutInterval, _ := time.ParseDuration(p.cfg.PollIntervalWorkout)
	sleepInterval, _ := time.ParseDuration(p.cfg.PollIntervalSleep)

	var wg sync.WaitGroup

	startLoop := func(name string, interval time.Duration, pollFunc func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.pollLoop(ctx, name, interval, pollFunc)
		}()
	}

	startLoop("cycles_recoveries", cycleInterval, p.pollCyclesAndRecoveries)
	startLoop("workouts", workoutInterval, p.pollWorkouts)
	startLoop("sleeps", sleepInterval, p.pollSleeps)

	wg.Wait()
}

func (p *Poller) pollLoop(ctx context.Context, name string, interval time.Duration, pollFunc func(ctx context.Context) error) {
	// Initial jitter to avoid thundering herd on startup
	jitter := time.Duration(rand.Intn(60)) * time.Second
	time.Sleep(jitter)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		start := time.Now()
		p.logger.Info("Starting poll run", "task", name)
		if err := pollFunc(ctx); err != nil {
			p.logger.Error("Poll run failed", "task", name, "error", err, "duration", time.Since(start))
			// We don't exit, we just wait for the next tick. Exponential backoff could be added for retries within the task.
		} else {
			p.logger.Info("Poll run succeeded", "task", name, "duration", time.Since(start))
		}

		select {
		case <-ctx.Done():
			p.logger.Info("Stopping poll loop", "task", name)
			return
		case <-ticker.C:
			// Continue to next run
		}
	}
}

func (p *Poller) RunAdHocSync(ctx context.Context, whoopUserID string) {
	p.logger.Info("Starting ad-hoc sync", "user_id", whoopUserID)

	// Since it's ad-hoc for a specific user, temporarily override or just pass down the user.
	// For simplicity, we just reuse the existing functions since they use p.whoopUserID.
	// In a real multi-tenant app, the functions would take whoopUserID as a param.

	if err := p.pollCyclesAndRecoveries(ctx); err != nil {
		p.logger.Error("Ad-hoc sync cycles failed", "error", err)
	}
	if err := p.pollWorkouts(ctx); err != nil {
		p.logger.Error("Ad-hoc sync workouts failed", "error", err)
	}
	if err := p.pollSleeps(ctx); err != nil {
		p.logger.Error("Ad-hoc sync sleeps failed", "error", err)
	}
	p.logger.Info("Ad-hoc sync completed", "user_id", whoopUserID)
}

func (p *Poller) pollCyclesAndRecoveries(ctx context.Context) error {
	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return err
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return err
	}

	// Respect global rate limit
	if err := p.limiter.Wait(ctx); err != nil {
		return err
	}

	cyclesPage, err := client.Cycle.List(ctx, nil)
	if err != nil {
		return err
	}

	for _, cycle := range cyclesPage.Records {
		// Use local variable for loop variable pointer issue avoidance is handled correctly in go1.22
		if err := p.storage.UpsertCycle(ctx, p.logger, internalUserID, &cycle); err != nil {
			p.logger.Error("Failed to upsert cycle", "id", cycle.ID, "err", err)
		}

		// Also fetch recovery for this cycle
		if err := p.limiter.Wait(ctx); err != nil {
			return err
		}

		recovery, err := client.Recovery.GetByID(ctx, cycle.ID)
		if err == nil && recovery != nil {
			if err := p.storage.UpsertRecovery(ctx, p.logger, internalUserID, recovery); err != nil {
				p.logger.Error("Failed to upsert recovery", "cycle_id", cycle.ID, "err", err)
			}
		}
	}
	return nil
}

func (p *Poller) pollWorkouts(ctx context.Context) error {
	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return err
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return err
	}

	if err := p.limiter.Wait(ctx); err != nil {
		return err
	}

	workoutsPage, err := client.Workout.List(ctx, nil)
	if err != nil {
		return err
	}

	for _, w := range workoutsPage.Records {
		if err := p.storage.UpsertWorkout(ctx, p.logger, internalUserID, &w); err != nil {
			p.logger.Error("Failed to upsert workout", "id", w.ID, "err", err)
		}
	}
	return nil
}

func (p *Poller) pollSleeps(ctx context.Context) error {
	hour := time.Now().Hour()
	if hour < 6 || hour > 12 {
		p.logger.Debug("Skipping sleep poll, outside 6AM-12PM window")
		return nil
	}

	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return err
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return err
	}

	if err := p.limiter.Wait(ctx); err != nil {
		return err
	}

	sleepsPage, err := client.Sleep.List(ctx, nil)
	if err != nil {
		return err
	}

	for _, s := range sleepsPage.Records {
		if err := p.storage.UpsertSleep(ctx, p.logger, internalUserID, &s); err != nil {
			p.logger.Error("Failed to upsert sleep", "id", s.ID, "err", err)
		}
	}
	return nil
}
