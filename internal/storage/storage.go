package storage

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/arvind/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
	db   *db.Queries
}

func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{
		pool: pool,
		db:   db.New(pool),
	}
}

func (s *Storage) DB() *db.Queries {
	return s.db
}

func (s *Storage) UpsertCycle(ctx context.Context, logger *slog.Logger, userID pgtype.UUID, cycle *whoop.Cycle) error {
	endTime := pgtype.Timestamptz{Valid: false}
	if cycle.End != nil && !cycle.End.IsZero() {
		endTime = pgtype.Timestamptz{Time: *cycle.End, Valid: true}
	}

	var timezoneOffset = ParseTimezoneOffset(cycle.TimezoneOffset)

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
	})
	if err != nil {
		logger.Error("Failed to upsert cycle", "id", cycle.ID, "err", err)
		return err
	}
	logger.Debug("Upserted cycle", "id", cycle.ID)
	return nil
}

func (s *Storage) UpsertSleep(ctx context.Context, logger *slog.Logger, userID pgtype.UUID, sleep *whoop.Sleep) error {
	endTime := pgtype.Timestamptz{Valid: false}
	if !sleep.End.IsZero() {
		endTime = pgtype.Timestamptz{Time: sleep.End, Valid: true}
	}

	var timezoneOffset = ParseTimezoneOffset(sleep.TimezoneOffset)

	var performance, respiratoryRate, sleepConsistency, sleepEfficiency pgtype.Float4
	var sleepDebt, totalInBed, totalAwake, totalNoData, totalLight, totalSlowWave, totalRem, sleepCycleCount, disturbanceCount pgtype.Int4

	if sleep.Score != nil {
		performance = pgtype.Float4{Float32: float32(sleep.Score.SleepPerformancePercentage), Valid: true}
		respiratoryRate = pgtype.Float4{Float32: float32(sleep.Score.RespiratoryRate), Valid: true}
		sleepConsistency = pgtype.Float4{Float32: float32(sleep.Score.SleepConsistencyPercentage), Valid: true}
		sleepEfficiency = pgtype.Float4{Float32: float32(sleep.Score.SleepEfficiencyPercentage), Valid: true}

		if sleep.Score.SleepNeeded != nil {
			sleepDebt = pgtype.Int4{Int32: int32(sleep.Score.SleepNeeded.NeedFromSleepDebtMilli), Valid: true}
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

	id, _ := strconv.ParseInt(sleep.ID, 10, 64)

	err := s.db.UpsertSleep(ctx, db.UpsertSleepParams{
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
	})
	if err != nil {
		logger.Error("Failed to upsert sleep", "id", sleep.ID, "err", err)
		return err
	}
	logger.Debug("Upserted sleep", "id", sleep.ID)
	return nil
}

func (s *Storage) UpsertRecovery(ctx context.Context, logger *slog.Logger, userID pgtype.UUID, recovery *whoop.Recovery) error {
	timezoneOffset := pgtype.Interval{Valid: false}

	var score, rhr, hrv, spo2, skinTemp pgtype.Float4

	if recovery.Score != nil {
		score = pgtype.Float4{Float32: float32(recovery.Score.RecoveryScore), Valid: true}
		rhr = pgtype.Float4{Float32: float32(recovery.Score.RestingHeartRate), Valid: true}
		hrv = pgtype.Float4{Float32: float32(recovery.Score.HrvRmssdMilli), Valid: true}
		spo2 = pgtype.Float4{Float32: float32(recovery.Score.Spo2Percentage), Valid: true}
		skinTemp = pgtype.Float4{Float32: float32(recovery.Score.SkinTempCelsius), Valid: true}
	}

	startTime := pgtype.Timestamptz{Time: recovery.CreatedAt, Valid: true}

	err := s.db.UpsertRecovery(ctx, db.UpsertRecoveryParams{
		ID:               int64(recovery.CycleID),
		UserID:           userID,
		StartTime:        startTime,
		TimezoneOffset:   timezoneOffset,
		RecoveryScore:    score,
		RestingHeartRate: rhr,
		HrvRmssdMilli:    hrv,
		Spo2Percentage:   spo2,
		SkinTempCelsius:  skinTemp,
	})
	if err != nil {
		logger.Error("Failed to upsert recovery", "id", recovery.CycleID, "err", err)
		return err
	}
	logger.Debug("Upserted recovery", "id", recovery.CycleID)
	return nil
}

func (s *Storage) UpsertWorkout(ctx context.Context, logger *slog.Logger, userID pgtype.UUID, workout *whoop.Workout) error {
	endTime := pgtype.Timestamptz{Valid: false}
	if !workout.End.IsZero() {
		endTime = pgtype.Timestamptz{Time: workout.End, Valid: true}
	}

	var timezoneOffset = ParseTimezoneOffset(workout.TimezoneOffset)

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

	id, _ := strconv.ParseInt(workout.ID, 10, 64)

	err := s.db.UpsertWorkout(ctx, db.UpsertWorkoutParams{
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
	})
	if err != nil {
		logger.Error("Failed to upsert workout", "id", workout.ID, "err", err)
		return err
	}
	logger.Debug("Upserted workout", "id", workout.ID)
	return nil
}
