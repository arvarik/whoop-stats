package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

// resetViper clears viper's global state between tests to avoid cross-test contamination.
func resetViper() {
	viper.Reset()
}


// setEnv is a helper that sets an env var and registers cleanup.
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	old, existed := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

// setRequiredEnv sets all required env vars to valid test values.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	setEnv(t, "WHOOP_STATS_DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	setEnv(t, "WHOOP_STATS_ENCRYPTION_KEY", "01234567890123456789012345678901") // 32 bytes
	setEnv(t, "WHOOP_STATS_WHOOP_CLIENT_ID", "test-client-id")
	setEnv(t, "WHOOP_STATS_WHOOP_CLIENT_SECRET", "test-client-secret")
}

func TestLoadConfig_Valid(t *testing.T) {
	resetViper()
	setRequiredEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.EncryptionKey != "01234567890123456789012345678901" {
		t.Errorf("unexpected encryption key: %s", cfg.EncryptionKey)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("unexpected database url: %s", cfg.DatabaseURL)
	}
	if cfg.WhoopClientID != "test-client-id" {
		t.Errorf("unexpected client id: %s", cfg.WhoopClientID)
	}
	// Check defaults
	if cfg.ServerPort != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.ServerPort)
	}
	if cfg.PollIntervalCycle != "4h" {
		t.Errorf("expected default poll interval 4h, got %s", cfg.PollIntervalCycle)
	}
}

func TestLoadConfig_MissingDatabaseURL(t *testing.T) {
	resetViper()
	setEnv(t, "WHOOP_STATS_DATABASE_URL", "")
	setEnv(t, "WHOOP_STATS_ENCRYPTION_KEY", "01234567890123456789012345678901")
	setEnv(t, "WHOOP_STATS_WHOOP_CLIENT_ID", "test")
	setEnv(t, "WHOOP_STATS_WHOOP_CLIENT_SECRET", "test")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing database URL")
	}
}

func TestLoadConfig_MissingEncryptionKey(t *testing.T) {
	resetViper()
	setRequiredEnv(t)
	setEnv(t, "WHOOP_STATS_ENCRYPTION_KEY", "")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing encryption key")
	}
}

func TestLoadConfig_WrongLengthEncryptionKey(t *testing.T) {
	resetViper()
	setRequiredEnv(t)
	setEnv(t, "WHOOP_STATS_ENCRYPTION_KEY", "too-short")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for wrong-length encryption key")
	}
}

func TestLoadConfig_MissingClientID(t *testing.T) {
	resetViper()
	setRequiredEnv(t)
	setEnv(t, "WHOOP_STATS_WHOOP_CLIENT_ID", "")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing client ID")
	}
}

func TestLoadConfig_MissingClientSecret(t *testing.T) {
	resetViper()
	setRequiredEnv(t)
	setEnv(t, "WHOOP_STATS_WHOOP_CLIENT_SECRET", "")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing client secret")
	}
}
