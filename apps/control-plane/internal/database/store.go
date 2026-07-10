package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"deltauptime/packages/database/postgres"
)

// Store owns the shared pgx pool and typed sqlc query set used by backend modules.
type Store struct {
	pool    *pgxpool.Pool
	Queries *postgres.Queries
}

// OpenStore creates a pgx pool, verifies connectivity, and wires sqlc queries.
func OpenStore(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pgx pool: %w", err)
	}

	return &Store{
		pool:    pool,
		Queries: postgres.New(pool),
	}, nil
}

func (s *Store) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

func (s *Store) Pool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	return s.pool
}
