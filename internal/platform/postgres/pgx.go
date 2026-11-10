package postgres

import (
	"context"
	"fmt"

	"github.com/emount4/udav-back/internal/core/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxExecutor interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

type pgxQueries struct {
	executor pgxExecutor
}

type pgxPool struct {
	pgxQueries
	pool *pgxpool.Pool
}

func (q pgxQueries) Exec(ctx context.Context, query string, args ...any) (db.Result, error) {
	result, err := q.executor.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("execute postgres query: %w", err)
	}
	return result, nil
}

func (q pgxQueries) Query(ctx context.Context, query string, args ...any) (db.Rows, error) {
	rows, err := q.executor.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query postgres rows: %w", err)
	}
	return pgxRows{rows: rows}, nil
}

func (q pgxQueries) QueryRow(ctx context.Context, query string, args ...any) db.Row {
	return pgxRow{row: q.executor.QueryRow(ctx, query, args...)}
}

func (p *pgxPool) Ping(ctx context.Context) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (p *pgxPool) Close() error {
	p.pool.Close()
	return nil
}

type pgxRow struct {
	row pgx.Row
}

func (r pgxRow) Scan(dest ...any) error {
	if err := r.row.Scan(dest...); err != nil {
		return fmt.Errorf("scan postgres row: %w", err)
	}
	return nil
}

type pgxRows struct {
	rows pgx.Rows
}

func (r pgxRows) Close() {
	r.rows.Close()
}

func (r pgxRows) Next() bool {
	return r.rows.Next()
}

func (r pgxRows) Scan(dest ...any) error {
	if err := r.rows.Scan(dest...); err != nil {
		return fmt.Errorf("scan postgres rows: %w", err)
	}
	return nil
}

func (r pgxRows) Err() error {
	if err := r.rows.Err(); err != nil {
		return fmt.Errorf("iterate postgres rows: %w", err)
	}
	return nil
}
