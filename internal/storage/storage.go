// Package storage maps WHOOP API domain objects to database records via sqlc.
// Every method performs an idempotent upsert (INSERT ... ON CONFLICT DO UPDATE)
// ensuring no duplicate data is ever recorded.
package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/arvarik/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Storage provides methods to persist WHOOP data into TimescaleDB.
type Storage struct {
	pool   *pgxpool.Pool
	db     *db.Queries
	logger *slog.Logger
}

// NewStorage creates a new Storage instance backed by the given connection pool.
func NewStorage(pool *pgxpool.Pool, logger *slog.Logger) *Storage {
	return &Storage{
		pool:   pool,
		db:     db.New(pool),
		logger: logger,
	}
}

// DB returns the underlying sqlc Queries instance for direct query access.
func (s *Storage) DB() *db.Queries {
	return s.db
}

// UpsertCycle persists a WHOOP cycle record, updating it if it already exists.
func (s *Storage) UpsertCycle(ctx context.Context, userID pgtype.UUID, cycle *whoop.Cycle) error {
	endTime := pgtype.Timestamptz{Valid: false}
	if cycle.End != nil && !cycle.End.IsZero() {
		endTime = pgtype.Timestamptz{Time: *cycle.End, Valid: true}
	}

	timezoneOffset := ParseTimezoneOffset(cycle.TimezoneOffset)

	var strain, kilojoule pgtype.Float4
	var avgHR, maxHR pgtype.Int4
	if cycle.Score != nil {
		strain = pgtype.Float4{Float32: float32(cycle.Score.Strain), Valid: true}
		kilojoule = pgtype.Float4{Float32: float32(cycle.Score.Kilojoule), Valid: true}
		avgHR = pgtype.Int4{Int32: int32(cycle.Score.AverageHeartRate), Valid: true}
		maxHR = pgtype.Int4{Int32: int32(cycle.Score.MaxHeartRate), Valid: true}
	}

	err := s.db.UpsertCycle(ctx, db.UpsertCycleParams{
		ID:               int64(cycle.ID),
		UserID:           userID,
		StartTime:        pgtype.Timestamptz{Time: cycle.Start, Valid: true},
		EndTime:          endTime,
		TimezoneOffset:   timezoneOffset,
		Strain:           strain,
		Kilojoule:        kilojoule,
		AverageHeartRate: avgHR,
		MaxHeartRate:     maxHR,
		ScoreState:       pgtype.Text{String: cycle.ScoreState, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("upserting cycle %d: %w", cycle.ID, err)
	}
	s.logger.Debug("Upserted cycle", "id", cycle.ID)
	return nil
}

// UpsertSleep persists a WHOOP sleep record with all stage and need data.
func (s *Storage) UpsertSleep(ctx context.Context, userID pgtype.UUID, sleep *whoop.Sleep) error {
	endTime := pgtype.Timestamptz{Valid: false}
	if !sleep.End.IsZero() {
		endTime = pgtype.Timestamptz{Time: sleep.End, Valid: true}
	}

	timezoneOffset := ParseTimezoneOffset(sleep.TimezoneOffset)

	var performance, respiratoryRate, sleepConsistency, sleepEfficiency pgtype.Float4
	var sleepDebt, totalInBed, totalAwake, totalNoData, totalLight, totalSlowWave, totalRem, sleepCycleCount, disturbanceCount pgtype.Int4
	var baseline, needStrain, needNap pgtype.Int4

	if sleep.Score != nil {
		performance = pgtype.Float4{Float32: float32(sleep.Score.SleepPerformancePercentage), Valid: true}
		respiratoryRate = pgtype.Float4{Float32: float32(sleep.Score.RespiratoryRate), Valid: true}
		sleepConsistency = pgtype.Float4{Float32: float32(sleep.Score.SleepConsistencyPercentage), Valid: true}
		sleepEfficiency = pgtype.Float4{Float32: float32(sleep.Score.SleepEfficiencyPercentage), Valid: true}

		if sleep.Score.SleepNeeded != nil {
			sleepDebt = pgtype.Int4{Int32: int32(sleep.Score.SleepNeeded.NeedFromSleepDebtMilli), Valid: true}
			baseline = pgtype.Int4{Int32: int32(sleep.Score.SleepNeeded.BaselineMilli), Valid: true}
			needStrain = pgtype.Int4{Int32: int32(sleep.Score.SleepNeeded.NeedFromRecentStrainMilli), Valid: true}
			needNap = pgtype.Int4{Int32: int32(sleep.Score.SleepNeeded.NeedFromRecentNapMilli), Valid: true}
		}

		if sleep.Score.StageSummary != nil {
			ss := sleep.Score.StageSummary
			totalInBed = pgtype.Int4{Int32: int32(ss.TotalInBedTimeMilli), Valid: true}
			totalAwake = pgtype.Int4{Int32: int32(ss.TotalAwakeTimeMilli), Valid: true}
			totalNoData = pgtype.Int4{Int32: int32(ss.TotalNoDataTimeMilli), Valid: true}
			totalLight = pgtype.Int4{Int32: int32(ss.TotalLightSleepTimeMilli), Valid: true}
			totalSlowWave = pgtype.Int4{Int32: int32(ss.TotalSlowWaveSleepTimeMilli), Valid: true}
			totalRem = pgtype.Int4{Int32: int32(ss.TotalRemSleepTimeMilli), Valid: true}
			sleepCycleCount = pgtype.Int4{Int32: int32(ss.SleepCycleCount), Valid: true}
			disturbanceCount = pgtype.Int4{Int32: int32(ss.DisturbanceCount), Valid: true}
		}
	}

	id, err := strconv.ParseInt(sleep.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing sleep ID %q: %w", sleep.ID, err)
	}

	if err := s.db.UpsertSleep(ctx, db.UpsertSleepParams{
		ID:                          id,
		UserID:                      userID,
		StartTime:                   pgtype.Timestamptz{Time: sleep.Start, Valid: true},
		EndTime:                     endTime,
		TimezoneOffset:              timezoneOffset,
		PerformanceScore:            performance,
		Nap:                         pgtype.Bool{Bool: sleep.Nap, Valid: true},
		RespiratoryRate:             respiratoryRate,
		SleepConsistencyPercentage:  sleepConsistency,
		SleepEfficiencyPercentage:   sleepEfficiency,
		SleepDebtMilli:              sleepDebt,
		TotalInBedTimeMilli:         totalInBed,
		TotalAwakeTimeMilli:         totalAwake,
		TotalNoDataTimeMilli:        totalNoData,
		TotalLightSleepTimeMilli:    totalLight,
		TotalSlowWaveSleepTimeMilli: totalSlowWave,
		TotalRemSleepTimeMilli:      totalRem,
		SleepCycleCount:             sleepCycleCount,
		DisturbanceCount:            disturbanceCount,
		CycleID:                     pgtype.Int8{Int64: int64(sleep.CycleID), Valid: true},
		ScoreState:                  pgtype.Text{String: sleep.ScoreState, Valid: true},
		BaselineMilli:               baseline,
		NeedFromRecentStrainMilli:   needStrain,
		NeedFromRecentNapMilli:      needNap,
	}); err != nil {
		return fmt.Errorf("upserting sleep %d: %w", id, err)
	}
	s.logger.Debug("Upserted sleep", "id", id)
	return nil
}

// UpsertRecovery persists a WHOOP recovery record.
func (s *Storage) UpsertRecovery(ctx context.Context, userID pgtype.UUID, recovery *whoop.Recovery) error {
	timezoneOffset := pgtype.Interval{Valid: false}

	var score, rhr, hrv, spo2, skinTemp pgtype.Float4
	var userCalibrating pgtype.Bool
	if recovery.Score != nil {
		score = pgtype.Float4{Float32: float32(recovery.Score.RecoveryScore), Valid: true}
		rhr = pgtype.Float4{Float32: float32(recovery.Score.RestingHeartRate), Valid: true}
		hrv = pgtype.Float4{Float32: float32(recovery.Score.HrvRmssdMilli), Valid: true}
		spo2 = pgtype.Float4{Float32: float32(recovery.Score.Spo2Percentage), Valid: true}
		skinTemp = pgtype.Float4{Float32: float32(recovery.Score.SkinTempCelsius), Valid: true}
		userCalibrating = pgtype.Bool{Bool: recovery.Score.UserCalibrating, Valid: true}
	}

	startTime := pgtype.Timestamptz{Time: recovery.CreatedAt, Valid: true}

	if err := s.db.UpsertRecovery(ctx, db.UpsertRecoveryParams{
		ID:               int64(recovery.CycleID),
		UserID:           userID,
		StartTime:        startTime,
		TimezoneOffset:   timezoneOffset,
		RecoveryScore:    score,
		RestingHeartRate: rhr,
		HrvRmssdMilli:    hrv,
		Spo2Percentage:   spo2,
		SkinTempCelsius:  skinTemp,
		SleepID:          pgtype.Text{String: recovery.SleepID, Valid: recovery.SleepID != ""},
		ScoreState:       pgtype.Text{String: recovery.ScoreState, Valid: true},
		UserCalibrating:  userCalibrating,
	}); err != nil {
		return fmt.Errorf("upserting recovery (cycle %d): %w", recovery.CycleID, err)
	}
	s.logger.Debug("Upserted recovery", "cycle_id", recovery.CycleID)
	return nil
}

// UpsertWorkout persists a WHOOP workout record with HR zones and GPS data.
func (s *Storage) UpsertWorkout(ctx context.Context, userID pgtype.UUID, workout *whoop.Workout) error {
	endTime := pgtype.Timestamptz{Valid: false}
	if !workout.End.IsZero() {
		endTime = pgtype.Timestamptz{Time: workout.End, Valid: true}
	}

	timezoneOffset := ParseTimezoneOffset(workout.TimezoneOffset)

	var strain, kilojoule, percentRecorded, distance, altGain, altChange pgtype.Float4
	var avgHR, maxHR, z0, z1, z2, z3, z4, z5 pgtype.Int4

	if workout.Score != nil {
		strain = pgtype.Float4{Float32: float32(workout.Score.Strain), Valid: true}
		kilojoule = pgtype.Float4{Float32: float32(workout.Score.Kilojoule), Valid: true}
		percentRecorded = pgtype.Float4{Float32: float32(workout.Score.PercentRecorded), Valid: true}
		avgHR = pgtype.Int4{Int32: int32(workout.Score.AverageHeartRate), Valid: true}
		maxHR = pgtype.Int4{Int32: int32(workout.Score.MaxHeartRate), Valid: true}

		if workout.Score.DistanceMeter != nil {
			distance = pgtype.Float4{Float32: float32(*workout.Score.DistanceMeter), Valid: true}
		}
		if workout.Score.AltitudeGainMeter != nil {
			altGain = pgtype.Float4{Float32: float32(*workout.Score.AltitudeGainMeter), Valid: true}
		}
		if workout.Score.AltitudeChangeMeter != nil {
			altChange = pgtype.Float4{Float32: float32(*workout.Score.AltitudeChangeMeter), Valid: true}
		}

		if workout.Score.ZoneDuration != nil {
			zd := workout.Score.ZoneDuration
			z0 = pgtype.Int4{Int32: int32(zd.ZoneZeroMilli), Valid: true}
			z1 = pgtype.Int4{Int32: int32(zd.ZoneOneMilli), Valid: true}
			z2 = pgtype.Int4{Int32: int32(zd.ZoneTwoMilli), Valid: true}
			z3 = pgtype.Int4{Int32: int32(zd.ZoneThreeMilli), Valid: true}
			z4 = pgtype.Int4{Int32: int32(zd.ZoneFourMilli), Valid: true}
			z5 = pgtype.Int4{Int32: int32(zd.ZoneFiveMilli), Valid: true}
		}
	}

	id, err := strconv.ParseInt(workout.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing workout ID %q: %w", workout.ID, err)
	}

	if err := s.db.UpsertWorkout(ctx, db.UpsertWorkoutParams{
		ID:                  id,
		UserID:              userID,
		StartTime:           pgtype.Timestamptz{Time: workout.Start, Valid: true},
		EndTime:             endTime,
		TimezoneOffset:      timezoneOffset,
		SportID:             pgtype.Int4{Int32: int32(workout.SportID), Valid: true},
		Strain:              strain,
		AverageHeartRate:    avgHR,
		MaxHeartRate:        maxHR,
		Kilojoule:           kilojoule,
		PercentRecorded:     percentRecorded,
		DistanceMeter:       distance,
		AltitudeGainMeter:   altGain,
		AltitudeChangeMeter: altChange,
		ZoneZeroMilli:       z0,
		ZoneOneMilli:        z1,
		ZoneTwoMilli:        z2,
		ZoneThreeMilli:      z3,
		ZoneFourMilli:       z4,
		ZoneFiveMilli:       z5,
		SportName:           pgtype.Text{String: workout.SportName, Valid: true},
		ScoreState:          pgtype.Text{String: workout.ScoreState, Valid: true},
	}); err != nil {
		return fmt.Errorf("upserting workout %d: %w", id, err)
	}
	s.logger.Debug("Upserted workout", "id", id)
	return nil
}

// UpsertUserProfile persists the user's basic WHOOP profile.
func (s *Storage) UpsertUserProfile(ctx context.Context, userID pgtype.UUID, profile *whoop.BasicProfile) error {
	err := s.db.UpsertUserProfile(ctx, db.UpsertUserProfileParams{
		ID:          userID,
		WhoopUserID: int64(profile.UserID),
		Email:       pgtype.Text{String: profile.Email, Valid: true},
		FirstName:   pgtype.Text{String: profile.FirstName, Valid: true},
		LastName:    pgtype.Text{String: profile.LastName, Valid: true},
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("upserting user profile: %w", err)
	}
	s.logger.Debug("Upserted user profile", "user_id", userID)
	return nil
}

// UpsertBodyMeasurement persists body measurement data (height, weight, max HR).
func (s *Storage) UpsertBodyMeasurement(ctx context.Context, userID pgtype.UUID, measurement *whoop.BodyMeasurement) error {
	err := s.db.UpsertBodyMeasurement(ctx, db.UpsertBodyMeasurementParams{
		ID:             userID,
		HeightMeter:    pgtype.Float4{Float32: float32(measurement.HeightMeter), Valid: true},
		WeightKilogram: pgtype.Float4{Float32: float32(measurement.WeightKilogram), Valid: true},
		MaxHeartRate:   pgtype.Int4{Int32: int32(measurement.MaxHeartRate), Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("upserting body measurement: %w", err)
	}
	s.logger.Debug("Upserted body measurement", "user_id", userID)
	return nil
}
