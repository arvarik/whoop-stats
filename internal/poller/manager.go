package poller

import (
	"context"
	"errors"
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

type Poller struct {
	cfg         *config.Config
	authManager *auth.Manager
	storage     *storage.Storage
	logger      *slog.Logger
	whoopUserID string

	// Rate limiter to respect API limits across all polling goroutines
	limiter *rate.Limiter

	lastOffpeakSleepPoll time.Time
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
	profileInterval, _ := time.ParseDuration(p.cfg.PollIntervalProfile)

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
	startLoop("profile", profileInterval, p.pollUserProfile)

	wg.Wait()
}

func (p *Poller) waitWithLog(ctx context.Context, task string) error {
	if p.limiter == nil {
		return nil
	}
	start := time.Now()
	if err := p.limiter.Wait(ctx); err != nil {
		return err
	}
	// If we waited more than 100ms over the expected interval, log it as a throttle event
	if d := time.Since(start); d > 600*time.Millisecond {
		p.logger.Debug("Poller throttled by local rate limiter", "task", task, "wait_duration", d)
	}
	return nil
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

	if err := p.pollCyclesAndRecoveries(ctx); err != nil {
		p.logger.Error("Ad-hoc sync cycles failed", "error", err)
	}
	if err := p.pollWorkouts(ctx); err != nil {
		p.logger.Error("Ad-hoc sync workouts failed", "error", err)
	}
	if err := p.pollSleeps(ctx); err != nil {
		p.logger.Error("Ad-hoc sync sleeps failed", "error", err)
	}
	if err := p.pollUserProfile(ctx); err != nil {
		p.logger.Error("Ad-hoc sync profile failed", "error", err)
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

	// Fetch cycles with pagination
	if err := p.waitWithLog(ctx, "cycles"); err != nil {
		return err
	}
	cyclesPage, err := client.Cycle.List(ctx, nil)
	pageCount := 0
	totalCycles := 0
	for {
		if err != nil {
			return err
		}
		pageCount++
		totalCycles += len(cyclesPage.Records)
		p.logger.Debug("Processing cycles page", "page", pageCount, "records", len(cyclesPage.Records))
		
		for _, cycle := range cyclesPage.Records {
			if err := p.storage.UpsertCycle(ctx, p.logger, internalUserID, &cycle); err != nil {
				p.logger.Error("Failed to upsert cycle", "id", cycle.ID, "err", err)
			}
		}
		cyclesPage, err = cyclesPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitWithLog(ctx, "cycles_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Cycles sync completed", "total_records", totalCycles, "pages", pageCount)

	// Fetch recoveries with pagination
	if err := p.waitWithLog(ctx, "recoveries"); err != nil {
		return err
	}
	recoveriesPage, err := client.Recovery.List(ctx, nil)
	pageCount = 0
	totalRecoveries := 0
	for {
		if err != nil {
			return err
		}
		pageCount++
		totalRecoveries += len(recoveriesPage.Records)
		p.logger.Debug("Processing recoveries page", "page", pageCount, "records", len(recoveriesPage.Records))

		for _, recovery := range recoveriesPage.Records {
			if err := p.storage.UpsertRecovery(ctx, p.logger, internalUserID, &recovery); err != nil {
				p.logger.Error("Failed to upsert recovery", "cycle_id", recovery.CycleID, "err", err)
			}
		}
		recoveriesPage, err = recoveriesPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitWithLog(ctx, "recoveries_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Recoveries sync completed", "total_records", totalRecoveries, "pages", pageCount)

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

	if err := p.waitWithLog(ctx, "workouts"); err != nil {
		return err
	}
	workoutsPage, err := client.Workout.List(ctx, nil)
	pageCount := 0
	totalWorkouts := 0
	for {
		if err != nil {
			return err
		}
		pageCount++
		totalWorkouts += len(workoutsPage.Records)
		p.logger.Debug("Processing workouts page", "page", pageCount, "records", len(workoutsPage.Records))

		for _, w := range workoutsPage.Records {
			if err := p.storage.UpsertWorkout(ctx, p.logger, internalUserID, &w); err != nil {
				p.logger.Error("Failed to upsert workout", "id", w.ID, "err", err)
			}
		}
		workoutsPage, err = workoutsPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitWithLog(ctx, "workouts_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Workouts sync completed", "total_records", totalWorkouts, "pages", pageCount)
	return nil
}

func (p *Poller) pollSleeps(ctx context.Context) error {
	hour := time.Now().Hour()
	isPeak := hour >= 6 && hour <= 12
	if !isPeak {
		offpeakInterval, _ := time.ParseDuration(p.cfg.PollIntervalSleepOffpeak)
		if time.Since(p.lastOffpeakSleepPoll) < offpeakInterval {
			p.logger.Debug("Skipping sleep poll, outside peak hours and offpeak interval not reached", "last_run", p.lastOffpeakSleepPoll)
			return nil
		}
		p.lastOffpeakSleepPoll = time.Now()
	}

	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return err
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return err
	}

	if err := p.waitWithLog(ctx, "sleeps"); err != nil {
		return err
	}
	sleepsPage, err := client.Sleep.List(ctx, nil)
	pageCount := 0
	totalSleeps := 0
	for {
		if err != nil {
			return err
		}
		pageCount++
		totalSleeps += len(sleepsPage.Records)
		p.logger.Debug("Processing sleeps page", "page", pageCount, "records", len(sleepsPage.Records))

		for _, s := range sleepsPage.Records {
			if err := p.storage.UpsertSleep(ctx, p.logger, internalUserID, &s); err != nil {
				p.logger.Error("Failed to upsert sleep", "id", s.ID, "err", err)
			}
		}
		sleepsPage, err = sleepsPage.NextPage(ctx)
		if errors.Is(err, whoop.ErrNoNextPage) {
			break
		}
		if err := p.waitWithLog(ctx, "sleeps_next"); err != nil {
			return err
		}
	}
	p.logger.Info("Sleeps sync completed", "total_records", totalSleeps, "pages", pageCount)
	return nil
}

func (p *Poller) pollUserProfile(ctx context.Context) error {
	client, err := p.authManager.GetClient(ctx, p.whoopUserID)
	if err != nil {
		return err
	}
	internalUserID, err := p.authManager.GetInternalUserID(ctx, p.whoopUserID)
	if err != nil {
		return err
	}

	if err := p.waitWithLog(ctx, "profile"); err != nil {
		return err
	}
	p.logger.Info("Fetching basic profile and body measurements")
	
	profile, err := client.User.GetBasicProfile(ctx)
	if err != nil {
		return err
	}
	if err := p.storage.UpsertUserProfile(ctx, p.logger, internalUserID, profile); err != nil {
		return err
	}

	if err := p.waitWithLog(ctx, "measurement"); err != nil {
		return err
	}
	measurement, err := client.User.GetBodyMeasurement(ctx)
	if err != nil {
		return err
	}
	if err := p.storage.UpsertBodyMeasurement(ctx, p.logger, internalUserID, measurement); err != nil {
		return err
	}

	p.logger.Info("User profile and measurements updated")
	return nil
}
