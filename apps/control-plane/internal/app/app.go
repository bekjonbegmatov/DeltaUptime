// Package app wires the uptime-server subcommands together. Keeping the dispatch
// logic here (rather than in main) makes it testable.
package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"deltauptime/apps/control-plane/internal/auth"
	"deltauptime/apps/control-plane/internal/config"
	"deltauptime/apps/control-plane/internal/database"
	"deltauptime/apps/control-plane/internal/httpapi"
)

// Version is the build version; overridden at release time via -ldflags.
var Version = "0.0.0-dev"

const usage = `uptime-server — DeltaUptime Control Plane

Usage:
  uptime-server <command>

Commands:
  api         Start the HTTP API + realtime gateway
  scheduler   Start the check scheduler
  worker      Start incident + notification workers
  migrate     Apply database migrations
  version     Print the version and exit
  help        Show this help
`

// Run dispatches a subcommand. out is where normal output goes (stdout in main).
func Run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		_, _ = fmt.Fprint(out, usage)
		return nil
	}

	cmd, rest := args[0], args[1:]
	log := newLogger()

	switch cmd {
	case "api":
		return runAPI(ctx, log)
	case "scheduler":
		return runPlaceholder(ctx, log, "scheduler")
	case "worker":
		return runPlaceholder(ctx, log, "worker")
	case "migrate":
		return runMigrate(ctx, log)
	case "version":
		_, _ = fmt.Fprintln(out, Version)
		return nil
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(out, usage)
		return nil
	default:
		_ = rest
		return fmt.Errorf("unknown command %q (try \"uptime-server help\")", cmd)
	}
}

func runAPI(ctx context.Context, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	srv := httpapi.NewServer(cfg.HTTPAddr, log)
	if cfg.PostgresDSN != "" {
		store, err := database.OpenStore(ctx, cfg.PostgresDSN)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer store.Close()

		authService, err := auth.NewService(store, auth.Config{
			AccessTokenSecret:  cfg.JWTAccessSecret,
			RefreshTokenSecret: cfg.JWTRefreshSecret,
			SecretsMasterKey:   cfg.SecretsMasterKey,
			AccessTokenTTL:     cfg.AccessTokenTTL,
			RefreshTokenTTL:    cfg.RefreshTokenTTL,
		})
		if err != nil {
			return fmt.Errorf("build auth service: %w", err)
		}

		srv = httpapi.NewServerWithAuth(cfg.HTTPAddr, log, auth.NewHandler(log, authService))
	} else {
		log.Warn("POSTGRES_DSN is empty: auth routes disabled, only health endpoints are served")
	}

	return httpapi.Serve(ctx, srv, log)
}

// runMigrate applies pending database migrations via goose.
func runMigrate(ctx context.Context, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return database.Migrate(ctx, cfg.PostgresDSN, log)
}

// runPlaceholder blocks until ctx is cancelled. Real scheduler/worker loops
// replace this in later phases.
func runPlaceholder(ctx context.Context, log *slog.Logger, name string) error {
	log.InfoContext(ctx, "starting (placeholder)", "component", name)
	<-ctx.Done()
	log.InfoContext(ctx, "shutting down", "component", name)
	return nil
}

func newLogger() *slog.Logger {
	return slog.Default()
}
