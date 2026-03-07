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

CREATE TABLE cycles (
    id BIGINT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    timezone_offset INTERVAL,
    strain REAL,
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
    strain REAL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, start_time)
);
SELECT create_hypertable('workouts', 'start_time');

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
