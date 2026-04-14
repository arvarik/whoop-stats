-- =============================================================================
-- WHOOP Stats — Initial Schema
-- =============================================================================
-- This migration creates the full schema for the whoop-stats application:
--   1. Core tables (users, tokens, profiles)
--   2. Time-series hypertables (cycles, recoveries, sleeps, workouts)
--   3. Webhook inbox for asynchronous event processing
--   4. Performance indexes for paginated queries
--   5. TimescaleDB continuous aggregates for dashboard roll-ups
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ---------------------------------------------------------------------------
-- Core User Tables
-- ---------------------------------------------------------------------------

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    whoop_user_id VARCHAR(255) UNIQUE NOT NULL,
    encrypted_access_token BYTEA NOT NULL,
    encrypted_refresh_token BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE users IS 'Core user accounts with AES-256-GCM encrypted OAuth2 tokens';

CREATE TABLE user_profiles (
    id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    whoop_user_id BIGINT UNIQUE NOT NULL,
    email TEXT,
    first_name TEXT,
    last_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE user_profiles IS 'WHOOP user profile data (name, email) fetched from /v1/user/profile/basic';

CREATE TABLE body_measurements (
    id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    height_meter REAL,
    weight_kilogram REAL,
    max_heart_rate INT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE body_measurements IS 'User body measurements from /v1/user/measurement/body';

-- ---------------------------------------------------------------------------
-- Webhook Inbox (store-then-process pattern for zero data loss)
-- ---------------------------------------------------------------------------

CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);
COMMENT ON TABLE webhook_events IS 'Webhook inbox: events stored immediately on receipt, processed asynchronously by the background worker';

-- ---------------------------------------------------------------------------
-- Time-Series Hypertables
-- ---------------------------------------------------------------------------
-- All time-series tables use composite PKs (id, start_time) required by
-- TimescaleDB hypertables for chunk-level uniqueness constraints.

CREATE TABLE cycles (
    id BIGINT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    timezone_offset INTERVAL,
    strain REAL,
    kilojoule REAL,
    average_heart_rate INT,
    max_heart_rate INT,
    score_state TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, start_time)
);
SELECT create_hypertable('cycles', 'start_time');
COMMENT ON TABLE cycles IS 'WHOOP physiological cycles (typically one per day) with strain scores';

CREATE TABLE recoveries (
    id BIGINT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    timezone_offset INTERVAL,
    recovery_score REAL,
    resting_heart_rate REAL,
    hrv_rmssd_milli REAL,
    spo2_percentage REAL,
    skin_temp_celsius REAL,
    sleep_id TEXT,
    score_state TEXT,
    user_calibrating BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, start_time)
);
SELECT create_hypertable('recoveries', 'start_time');
COMMENT ON TABLE recoveries IS 'Daily recovery scores with HRV, RHR, SpO2, and skin temperature';

CREATE TABLE sleeps (
    id TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    timezone_offset INTERVAL,
    performance_score REAL,
    nap BOOLEAN,
    respiratory_rate REAL,
    sleep_consistency_percentage REAL,
    sleep_efficiency_percentage REAL,
    sleep_debt_milli INT,
    total_in_bed_time_milli INT,
    total_awake_time_milli INT,
    total_no_data_time_milli INT,
    total_light_sleep_time_milli INT,
    total_slow_wave_sleep_time_milli INT,
    total_rem_sleep_time_milli INT,
    sleep_cycle_count INT,
    disturbance_count INT,
    cycle_id BIGINT,
    score_state TEXT,
    baseline_milli INT,
    need_from_recent_strain_milli INT,
    need_from_recent_nap_milli INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, start_time)
);
SELECT create_hypertable('sleeps', 'start_time');
COMMENT ON TABLE sleeps IS 'Sleep sessions with stage breakdowns, sleep need, and debt tracking';

CREATE TABLE workouts (
    id TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    timezone_offset INTERVAL,
    sport_id INT,
    sport_name TEXT,
    strain REAL,
    average_heart_rate INT,
    max_heart_rate INT,
    kilojoule REAL,
    percent_recorded REAL,
    distance_meter REAL,
    altitude_gain_meter REAL,
    altitude_change_meter REAL,
    zone_zero_milli INT,
    zone_one_milli INT,
    zone_two_milli INT,
    zone_three_milli INT,
    zone_four_milli INT,
    zone_five_milli INT,
    score_state TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, start_time)
);
SELECT create_hypertable('workouts', 'start_time');
COMMENT ON TABLE workouts IS 'Workout sessions with HR zones, GPS data, and sport classification';

-- ---------------------------------------------------------------------------
-- Performance Indexes
-- ---------------------------------------------------------------------------
-- Composite indexes on (user_id, start_time DESC) accelerate the paginated
-- GetX queries that filter by user and order by recency.

CREATE INDEX IF NOT EXISTS idx_cycles_user_start ON cycles(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sleeps_user_start ON sleeps(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_workouts_user_start ON workouts(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_recoveries_user_start ON recoveries(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_status_created ON webhook_events(status, created_at ASC);

-- ---------------------------------------------------------------------------
-- Continuous Aggregates (materialized views auto-refreshed by TimescaleDB)
-- ---------------------------------------------------------------------------
-- These power the /insights endpoint and 30-day trend charts on the dashboard.
-- Refresh policies ensure data stays current (checked hourly, looking back 3 days).

CREATE MATERIALIZED VIEW daily_strain
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 day', start_time) AS bucket,
    AVG(strain) AS avg_strain,
    MAX(strain) AS max_strain
FROM cycles
GROUP BY user_id, time_bucket('1 day', start_time);

SELECT add_continuous_aggregate_policy('daily_strain',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

CREATE MATERIALIZED VIEW weekly_strain
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 week', start_time) AS bucket,
    AVG(strain) AS avg_strain,
    MAX(strain) AS max_strain
FROM cycles
GROUP BY user_id, time_bucket('1 week', start_time);

SELECT add_continuous_aggregate_policy('weekly_strain',
    start_offset => INTERVAL '1 month',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

CREATE MATERIALIZED VIEW daily_recovery
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 day', start_time) AS bucket,
    AVG(recovery_score) AS avg_recovery
FROM recoveries
GROUP BY user_id, time_bucket('1 day', start_time);

SELECT add_continuous_aggregate_policy('daily_recovery',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

CREATE MATERIALIZED VIEW weekly_recovery
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 week', start_time) AS bucket,
    AVG(recovery_score) AS avg_recovery
FROM recoveries
GROUP BY user_id, time_bucket('1 week', start_time);

SELECT add_continuous_aggregate_policy('weekly_recovery',
    start_offset => INTERVAL '1 month',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

CREATE MATERIALIZED VIEW daily_sleep
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 day', start_time) AS bucket,
    AVG(performance_score) AS avg_performance,
    AVG(sleep_efficiency_percentage) AS avg_efficiency
FROM sleeps
WHERE nap = false
GROUP BY user_id, time_bucket('1 day', start_time);

SELECT add_continuous_aggregate_policy('daily_sleep',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');
