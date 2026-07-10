// Package config loads Control Plane configuration from the environment.
// Keep this dependency-free (stdlib only) so it stays trivial to test.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds runtime configuration for uptime-server. Fields mirror .env.example.
type Config struct {
	AppEnv           string        // APP_ENV
	HTTPAddr         string        // HTTP_ADDR
	PostgresDSN      string        // POSTGRES_DSN
	RedisURL         string        // REDIS_URL
	NATSURL          string        // NATS_URL
	LogLevel         string        // LOG_LEVEL
	JWTAccessSecret  string        // JWT_ACCESS_SECRET
	JWTRefreshSecret string        // JWT_REFRESH_SECRET
	SecretsMasterKey string        // SECRETS_MASTER_KEY
	AccessTokenTTL   time.Duration // ACCESS_TOKEN_TTL
	RefreshTokenTTL  time.Duration // REFRESH_TOKEN_TTL
}

// Load reads configuration from the environment, applying sane defaults for
// local development. It never fails today, but returns error for forward
// compatibility once required fields are validated.
func Load() (Config, error) {
	appEnv := getenv("APP_ENV", "development")

	accessTTL, err := parseDurationEnv("ACCESS_TOKEN_TTL", "15m")
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := parseDurationEnv("REFRESH_TOKEN_TTL", "720h")
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:           appEnv,
		HTTPAddr:         getenv("HTTP_ADDR", ":8080"),
		PostgresDSN:      os.Getenv("POSTGRES_DSN"),
		RedisURL:         os.Getenv("REDIS_URL"),
		NATSURL:          os.Getenv("NATS_URL"),
		LogLevel:         getenv("LOG_LEVEL", "info"),
		JWTAccessSecret:  getenv("JWT_ACCESS_SECRET", "dev-access-secret-not-for-production"),
		JWTRefreshSecret: getenv("JWT_REFRESH_SECRET", "dev-refresh-secret-not-for-production"),
		SecretsMasterKey: getenv("SECRETS_MASTER_KEY", "dev-secrets-master-key-not-for-production"),
		AccessTokenTTL:   accessTTL,
		RefreshTokenTTL:  refreshTTL,
	}

	if appEnv != "development" {
		if cfg.JWTAccessSecret == "dev-access-secret-not-for-production" {
			return Config{}, fmt.Errorf("JWT_ACCESS_SECRET is required outside development")
		}
		if cfg.JWTRefreshSecret == "dev-refresh-secret-not-for-production" {
			return Config{}, fmt.Errorf("JWT_REFRESH_SECRET is required outside development")
		}
		if cfg.SecretsMasterKey == "dev-secrets-master-key-not-for-production" {
			return Config{}, fmt.Errorf("SECRETS_MASTER_KEY is required outside development")
		}
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseDurationEnv(key, fallback string) (time.Duration, error) {
	raw := getenv(key, fallback)
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: parse duration: %w", key, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return value, nil
}
