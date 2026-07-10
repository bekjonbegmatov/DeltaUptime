// Package database owns low-level database access for the Control Plane: migrations
// now, and (later) the pgx connection pool shared by the modules.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx"
	"github.com/pressly/goose/v3"

	"deltauptime/migrations"
)

// Migrate applies all pending migrations against the given Postgres DSN.
// It is safe to run repeatedly: goose only applies migrations not yet recorded.
func Migrate(ctx context.Context, dsn string, log *slog.Logger) error {
	if dsn == "" {
		return fmt.Errorf("POSTGRES_DSN is required for migrate")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(gooseLogger{log})
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	log.InfoContext(ctx, "migrations applied")
	return nil
}

// gooseLogger adapts slog to goose's logger interface.
type gooseLogger struct{ log *slog.Logger }

func (g gooseLogger) Fatalf(format string, v ...any) { g.log.Error(fmt.Sprintf(format, v...)) }
func (g gooseLogger) Printf(format string, v ...any) { g.log.Info(fmt.Sprintf(format, v...)) }
