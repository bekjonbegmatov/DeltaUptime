package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure a clean environment for the keys we care about.
	for _, k := range []string{"APP_ENV", "HTTP_ADDR", "LOG_LEVEL", "POSTGRES_DSN", "ACCESS_TOKEN_TTL", "REFRESH_TOKEN_TTL"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want development", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %s, want 15m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 720*time.Hour {
		t.Errorf("RefreshTokenTTL = %s, want 720h", cfg.RefreshTokenTTL)
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("POSTGRES_DSN", "postgres://x")
	t.Setenv("JWT_ACCESS_SECRET", "access")
	t.Setenv("JWT_REFRESH_SECRET", "refresh")
	t.Setenv("ACCESS_TOKEN_TTL", "30m")
	t.Setenv("REFRESH_TOKEN_TTL", "24h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != "production" {
		t.Errorf("AppEnv = %q, want production", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":9000" {
		t.Errorf("HTTPAddr = %q, want :9000", cfg.HTTPAddr)
	}
	if cfg.PostgresDSN != "postgres://x" {
		t.Errorf("PostgresDSN = %q, want postgres://x", cfg.PostgresDSN)
	}
	if cfg.AccessTokenTTL != 30*time.Minute {
		t.Errorf("AccessTokenTTL = %s, want 30m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 24*time.Hour {
		t.Errorf("RefreshTokenTTL = %s, want 24h", cfg.RefreshTokenTTL)
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid ACCESS_TOKEN_TTL")
	}
}
