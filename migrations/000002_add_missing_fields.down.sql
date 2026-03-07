-- Performance Indexes
DROP INDEX IF EXISTS idx_webhook_status_created;
DROP INDEX IF EXISTS idx_recoveries_user_start;
DROP INDEX IF EXISTS idx_workouts_user_start;
DROP INDEX IF EXISTS idx_sleeps_user_start;
DROP INDEX IF EXISTS idx_cycles_user_start;

ALTER TABLE workouts 
DROP COLUMN average_heart_rate, DROP COLUMN max_heart_rate, DROP COLUMN kilojoule, DROP COLUMN percent_recorded, DROP COLUMN distance_meter, DROP COLUMN altitude_gain_meter, DROP COLUMN altitude_change_meter, DROP COLUMN zone_zero_milli, DROP COLUMN zone_one_milli, DROP COLUMN zone_two_milli, DROP COLUMN zone_three_milli, DROP COLUMN zone_four_milli, DROP COLUMN zone_five_milli;

ALTER TABLE recoveries
DROP COLUMN resting_heart_rate, DROP COLUMN hrv_rmssd_milli, DROP COLUMN spo2_percentage, DROP COLUMN skin_temp_celsius;

ALTER TABLE sleeps
DROP COLUMN nap, DROP COLUMN respiratory_rate, DROP COLUMN sleep_consistency_percentage, DROP COLUMN sleep_efficiency_percentage, DROP COLUMN sleep_debt_milli, DROP COLUMN total_in_bed_time_milli, DROP COLUMN total_awake_time_milli, DROP COLUMN total_no_data_time_milli, DROP COLUMN total_light_sleep_time_milli, DROP COLUMN total_slow_wave_sleep_time_milli, DROP COLUMN total_rem_sleep_time_milli, DROP COLUMN sleep_cycle_count, DROP COLUMN disturbance_count;

ALTER TABLE cycles
DROP COLUMN kilojoule, DROP COLUMN average_heart_rate, DROP COLUMN max_heart_rate;
