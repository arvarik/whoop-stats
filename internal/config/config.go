package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL        string `mapstructure:"DATABASE_URL"`
	ServerPort         string `mapstructure:"SERVER_PORT"`
	EncryptionKey      string `mapstructure:"ENCRYPTION_KEY"` // Must be 32 bytes for AES-256
	WhoopClientID      string `mapstructure:"WHOOP_CLIENT_ID"`
	WhoopClientSecret  string `mapstructure:"WHOOP_CLIENT_SECRET"`
	WhoopWebhookSecret string `mapstructure:"WHOOP_WEBHOOK_SECRET"`

	// Polling Intervals
	PollIntervalCycle   string `mapstructure:"POLL_INTERVAL_CYCLE"`
	PollIntervalWorkout string `mapstructure:"POLL_INTERVAL_WORKOUT"`
	PollIntervalSleep   string `mapstructure:"POLL_INTERVAL_SLEEP"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	viper.SetEnvPrefix("WHOOP_STATS")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults suitable for local development
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DATABASE_URL", "postgres://whoop_user:secretpassword@localhost:5432/whoop_stats?sslmode=disable")
	viper.SetDefault("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef") // 32 bytes for AES-256
	
	viper.SetDefault("POLL_INTERVAL_CYCLE", "4h")
	viper.SetDefault("POLL_INTERVAL_WORKOUT", "30m")
	viper.SetDefault("POLL_INTERVAL_SLEEP", "1h")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	if len(config.EncryptionKey) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes for AES-256")
	}

	return &config, nil
}
