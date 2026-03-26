// Package config provides application configuration loaded from environment
// variables with the WHOOP_STATS_ prefix. It supports sensible defaults for
// local development while requiring critical secrets to be explicitly set.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application settings. Values map to environment variables
// prefixed with WHOOP_STATS_ (e.g. WHOOP_STATS_DATABASE_URL).
type Config struct {
	DatabaseURL        string `mapstructure:"DATABASE_URL"`
	ServerPort         string `mapstructure:"SERVER_PORT"`
	EncryptionKey      string `mapstructure:"ENCRYPTION_KEY"` // Must be exactly 32 bytes for AES-256-GCM
	WhoopClientID      string `mapstructure:"WHOOP_CLIENT_ID"`
	WhoopClientSecret  string `mapstructure:"WHOOP_CLIENT_SECRET"`
	WhoopWebhookSecret string `mapstructure:"WHOOP_WEBHOOK_SECRET"`

	CorsAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`

	LogLevel string `mapstructure:"LOG_LEVEL"`

	// Polling intervals (Go duration strings, e.g. "4h", "30m")
	PollIntervalCycle        string `mapstructure:"POLL_INTERVAL_CYCLE"`
	PollIntervalWorkout      string `mapstructure:"POLL_INTERVAL_WORKOUT"`
	PollIntervalSleep        string `mapstructure:"POLL_INTERVAL_SLEEP"`
	PollIntervalSleepOffpeak string `mapstructure:"POLL_INTERVAL_SLEEP_OFFPEAK"`
	PollIntervalProfile      string `mapstructure:"POLL_INTERVAL_PROFILE"`
}

// LoadConfig reads configuration from environment variables prefixed with WHOOP_STATS_.
// It returns an error if required fields are missing or invalid.
func LoadConfig() (*Config, error) {
	viper.SetEnvPrefix("WHOOP_STATS")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults suitable for local development
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ENCRYPTION_KEY", "")
	viper.SetDefault("WHOOP_CLIENT_ID", "")
	viper.SetDefault("WHOOP_CLIENT_SECRET", "")
	viper.SetDefault("WHOOP_WEBHOOK_SECRET", "")

	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3032")

	viper.SetDefault("POLL_INTERVAL_CYCLE", "4h")
	viper.SetDefault("POLL_INTERVAL_WORKOUT", "30m")
	viper.SetDefault("POLL_INTERVAL_SLEEP", "1h")
	viper.SetDefault("POLL_INTERVAL_SLEEP_OFFPEAK", "4h")
	viper.SetDefault("POLL_INTERVAL_PROFILE", "24h")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Validate required fields
	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("WHOOP_STATS_DATABASE_URL is required (e.g. postgres://user:pass@localhost:5432/dbname)")
	}

	if config.EncryptionKey == "" {
		return nil, fmt.Errorf("WHOOP_STATS_ENCRYPTION_KEY is required (32 bytes for AES-256, generate with: openssl rand -hex 16)")
	}
	if len(config.EncryptionKey) != 32 {
		return nil, fmt.Errorf("WHOOP_STATS_ENCRYPTION_KEY must be exactly 32 bytes for AES-256 (got %d bytes)", len(config.EncryptionKey))
	}

	if config.WhoopClientID == "" {
		return nil, fmt.Errorf("WHOOP_STATS_WHOOP_CLIENT_ID is required (from https://developer.whoop.com)")
	}
	if config.WhoopClientSecret == "" {
		return nil, fmt.Errorf("WHOOP_STATS_WHOOP_CLIENT_SECRET is required (from https://developer.whoop.com)")
	}

	return &config, nil
}
