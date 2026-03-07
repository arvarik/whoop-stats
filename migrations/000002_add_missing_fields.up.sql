ALTER TABLE cycles 
ADD COLUMN kilojoule REAL,
ADD COLUMN average_heart_rate INT,
ADD COLUMN max_heart_rate INT;

ALTER TABLE sleeps
ADD COLUMN nap BOOLEAN,
ADD COLUMN respiratory_rate REAL,
ADD COLUMN sleep_consistency_percentage REAL,
ADD COLUMN sleep_efficiency_percentage REAL,
ADD COLUMN sleep_debt_milli INT,
ADD COLUMN total_in_bed_time_milli INT,
ADD COLUMN total_awake_time_milli INT,
ADD COLUMN total_no_data_time_milli INT,
ADD COLUMN total_light_sleep_time_milli INT,
ADD COLUMN total_slow_wave_sleep_time_milli INT,
ADD COLUMN total_rem_sleep_time_milli INT,
ADD COLUMN sleep_cycle_count INT,
ADD COLUMN disturbance_count INT;

ALTER TABLE recoveries
ADD COLUMN resting_heart_rate REAL,
ADD COLUMN hrv_rmssd_milli REAL,
ADD COLUMN spo2_percentage REAL,
ADD COLUMN skin_temp_celsius REAL;

ALTER TABLE workouts
ADD COLUMN average_heart_rate INT,
ADD COLUMN max_heart_rate INT,
ADD COLUMN kilojoule REAL,
ADD COLUMN percent_recorded REAL,
ADD COLUMN distance_meter REAL,
ADD COLUMN altitude_gain_meter REAL,
ADD COLUMN altitude_change_meter REAL,
ADD COLUMN zone_zero_milli INT,
ADD COLUMN zone_one_milli INT,
ADD COLUMN zone_two_milli INT,
ADD COLUMN zone_three_milli INT,
ADD COLUMN zone_four_milli INT,
ADD COLUMN zone_five_milli INT;

-- Performance Indexes
CREATE INDEX IF NOT EXISTS idx_cycles_user_start ON cycles(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sleeps_user_start ON sleeps(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_workouts_user_start ON workouts(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_recoveries_user_start ON recoveries(user_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_status_created ON webhook_events(status, created_at ASC);
