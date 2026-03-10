CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    whoop_user_id VARCHAR(255) UNIQUE NOT NULL,
    encrypted_access_token BYTEA NOT NULL,
    encrypted_refresh_token BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_profiles (
    id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    whoop_user_id BIGINT UNIQUE NOT NULL,
    email TEXT,
    first_name TEXT,
    last_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE body_measurements (
    id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    height_meter REAL,
    weight_kilogram REAL,
    max_heart_rate INT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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
    sleep_id BIGINT,
    score_state TEXT,
    user_calibrating BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, start_time)
);
SELECT create_hypertable('recoveries', 'start_time');

CREATE TABLE sleeps (
    id BIGINT NOT NULL,
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

CREATE TABLE workouts (
    id BIGINT NOT NULL,
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

-- Performance Indexes
CREATE INDEX IF NOT EXISTS idx_cycles_user_start ON cycles(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sleeps_user_start ON sleeps(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_workouts_user_start ON workouts(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_recoveries_user_start ON recoveries(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_status_created ON webhook_events(status, created_at ASC);

-- Continuous Aggregates for daily and weekly roll-ups
CREATE MATERIALIZED VIEW daily_strain
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 day', start_time) AS bucket,
    AVG(strain) AS avg_strain,
    MAX(strain) AS max_strain
FROM cycles
GROUP BY user_id, time_bucket('1 day', start_time);

CREATE MATERIALIZED VIEW weekly_strain
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 week', start_time) AS bucket,
    AVG(strain) AS avg_strain,
    MAX(strain) AS max_strain
FROM cycles
GROUP BY user_id, time_bucket('1 week', start_time);

CREATE MATERIALIZED VIEW daily_recovery
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 day', start_time) AS bucket,
    AVG(recovery_score) AS avg_recovery
FROM recoveries
GROUP BY user_id, time_bucket('1 day', start_time);

CREATE MATERIALIZED VIEW weekly_recovery
WITH (timescaledb.continuous) AS
SELECT
    user_id,
    time_bucket('1 week', start_time) AS bucket,
    AVG(recovery_score) AS avg_recovery
FROM recoveries
GROUP BY user_id, time_bucket('1 week', start_time);
