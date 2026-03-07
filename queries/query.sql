-- name: UpsertUser :one
INSERT INTO users (whoop_user_id, encrypted_access_token, encrypted_refresh_token, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (whoop_user_id) DO UPDATE SET
    encrypted_access_token = EXCLUDED.encrypted_access_token,
    encrypted_refresh_token = EXCLUDED.encrypted_refresh_token,
    updated_at = NOW()
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByWhoopID :one
SELECT * FROM users
WHERE whoop_user_id = $1 LIMIT 1;

-- name: UpsertCycle :exec
INSERT INTO cycles (id, user_id, start_time, end_time, timezone_offset, strain, kilojoule, average_heart_rate, max_heart_rate, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
ON CONFLICT (id, start_time) DO UPDATE SET
    end_time = EXCLUDED.end_time,
    timezone_offset = EXCLUDED.timezone_offset,
    strain = EXCLUDED.strain,
    kilojoule = EXCLUDED.kilojoule,
    average_heart_rate = EXCLUDED.average_heart_rate,
    max_heart_rate = EXCLUDED.max_heart_rate,
    updated_at = NOW();

-- name: UpsertRecovery :exec
INSERT INTO recoveries (id, user_id, start_time, timezone_offset, recovery_score, resting_heart_rate, hrv_rmssd_milli, spo2_percentage, skin_temp_celsius, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
ON CONFLICT (id, start_time) DO UPDATE SET
    timezone_offset = EXCLUDED.timezone_offset,
    recovery_score = EXCLUDED.recovery_score,
    resting_heart_rate = EXCLUDED.resting_heart_rate,
    hrv_rmssd_milli = EXCLUDED.hrv_rmssd_milli,
    spo2_percentage = EXCLUDED.spo2_percentage,
    skin_temp_celsius = EXCLUDED.skin_temp_celsius,
    updated_at = NOW();

-- name: UpsertSleep :exec
INSERT INTO sleeps (id, user_id, start_time, end_time, timezone_offset, performance_score, nap, respiratory_rate, sleep_consistency_percentage, sleep_efficiency_percentage, sleep_debt_milli, total_in_bed_time_milli, total_awake_time_milli, total_no_data_time_milli, total_light_sleep_time_milli, total_slow_wave_sleep_time_milli, total_rem_sleep_time_milli, sleep_cycle_count, disturbance_count, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW(), NOW())
ON CONFLICT (id, start_time) DO UPDATE SET
    end_time = EXCLUDED.end_time,
    timezone_offset = EXCLUDED.timezone_offset,
    performance_score = EXCLUDED.performance_score,
    nap = EXCLUDED.nap,
    respiratory_rate = EXCLUDED.respiratory_rate,
    sleep_consistency_percentage = EXCLUDED.sleep_consistency_percentage,
    sleep_efficiency_percentage = EXCLUDED.sleep_efficiency_percentage,
    sleep_debt_milli = EXCLUDED.sleep_debt_milli,
    total_in_bed_time_milli = EXCLUDED.total_in_bed_time_milli,
    total_awake_time_milli = EXCLUDED.total_awake_time_milli,
    total_no_data_time_milli = EXCLUDED.total_no_data_time_milli,
    total_light_sleep_time_milli = EXCLUDED.total_light_sleep_time_milli,
    total_slow_wave_sleep_time_milli = EXCLUDED.total_slow_wave_sleep_time_milli,
    total_rem_sleep_time_milli = EXCLUDED.total_rem_sleep_time_milli,
    sleep_cycle_count = EXCLUDED.sleep_cycle_count,
    disturbance_count = EXCLUDED.disturbance_count,
    updated_at = NOW();

-- name: UpsertWorkout :exec
INSERT INTO workouts (id, user_id, start_time, end_time, timezone_offset, sport_id, strain, average_heart_rate, max_heart_rate, kilojoule, percent_recorded, distance_meter, altitude_gain_meter, altitude_change_meter, zone_zero_milli, zone_one_milli, zone_two_milli, zone_three_milli, zone_four_milli, zone_five_milli, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), NOW())
ON CONFLICT (id, start_time) DO UPDATE SET
    end_time = EXCLUDED.end_time,
    timezone_offset = EXCLUDED.timezone_offset,
    sport_id = EXCLUDED.sport_id,
    strain = EXCLUDED.strain,
    average_heart_rate = EXCLUDED.average_heart_rate,
    max_heart_rate = EXCLUDED.max_heart_rate,
    kilojoule = EXCLUDED.kilojoule,
    percent_recorded = EXCLUDED.percent_recorded,
    distance_meter = EXCLUDED.distance_meter,
    altitude_gain_meter = EXCLUDED.altitude_gain_meter,
    altitude_change_meter = EXCLUDED.altitude_change_meter,
    zone_zero_milli = EXCLUDED.zone_zero_milli,
    zone_one_milli = EXCLUDED.zone_one_milli,
    zone_two_milli = EXCLUDED.zone_two_milli,
    zone_three_milli = EXCLUDED.zone_three_milli,
    zone_four_milli = EXCLUDED.zone_four_milli,
    zone_five_milli = EXCLUDED.zone_five_milli,
    updated_at = NOW();

-- name: CreateWebhookEvent :one
INSERT INTO webhook_events (payload, status, created_at)
VALUES ($1, $2, NOW())
RETURNING *;

-- name: GetPendingWebhookEvents :many
SELECT * FROM webhook_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: UpdateWebhookEventStatus :exec
UPDATE webhook_events
SET status = $2
WHERE id = $1;

-- name: GetCycles :many
SELECT * FROM cycles
WHERE user_id = $1 AND start_time < $2
ORDER BY start_time DESC
LIMIT $3;

-- name: GetSleeps :many
SELECT * FROM sleeps
WHERE user_id = $1 AND start_time < $2
ORDER BY start_time DESC
LIMIT $3;

-- name: GetWorkouts :many
SELECT * FROM workouts
WHERE user_id = $1 AND start_time < $2
ORDER BY start_time DESC
LIMIT $3;

-- name: GetDailyStrain :many
SELECT * FROM daily_strain
WHERE user_id = $1 AND bucket >= $2
ORDER BY bucket ASC;

-- name: GetDailyRecovery :many
SELECT * FROM daily_recovery
WHERE user_id = $1 AND bucket >= $2
ORDER BY bucket ASC;

