// Package config loads Control Plane configuration from the environment.
// Keep this dependency-free (stdlib only) so it stays trivial to test.
package config

import "os"

// Config holds runtime configuration for uptime-server. Fields mirror .env.example.
type Config struct {
	AppEnv      string // APP_ENV
	HTTPAddr    string // HTTP_ADDR
	PostgresDSN string // POSTGRES_DSN
	RedisURL    string // REDIS_URL
	NATSURL     string // NATS_URL
	LogLevel    string // LOG_LEVEL
}

// Load reads configuration from the environment, applying sane defaults for
// local development. It never fails today, but returns error for forward
// compatibility once required fields are validated.
func Load() (Config, error) {
	return Config{
		AppEnv:      getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		PostgresDSN: os.Getenv("POSTGRES_DSN"),
		RedisURL:    os.Getenv("REDIS_URL"),
		NATSURL:     os.Getenv("NATS_URL"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
	}, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
