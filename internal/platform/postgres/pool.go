package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/emount4/udav-back/internal/core/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MustOpen(ctx context.Context, cfg Config) db.Pool {
	pool, err := Open(ctx, cfg)
	if err != nil {
		panic(err)
	}
	return pool
}

func Open(ctx context.Context, cfg Config) (db.Pool, error) {
	if !cfg.Enabled() {
		return nil, errors.New("DATABASE_URL is required to open PostgreSQL")
	}

	ctx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	driverPool, err := pgxpool.NewWithConfig(ctx, cfg.pool.Copy())
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	pool := &pgxPool{pgxQueries: pgxQueries{executor: driverPool}, pool: driverPool}
	if err := pool.Ping(ctx); err != nil {
		driverPool.Close()
		return nil, err
	}
	return pool, nil
}
