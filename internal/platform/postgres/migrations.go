package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

func ApplyMigrations(ctx context.Context, cfg Config, migrations fs.FS) (_ int, err error) {
	if !cfg.Enabled() {
		return 0, errors.New("DATABASE_URL is required to apply PostgreSQL migrations")
	}
	// Goose ищет файлы по шаблону, поэтому отсутствие каталога проверяем отдельно.
	if _, err := fs.ReadDir(migrations, "."); err != nil {
		return 0, fmt.Errorf("read PostgreSQL migrations directory: %w", err)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return 0, fmt.Errorf("create PostgreSQL migration lock: %w", err)
	}
	db := stdlib.OpenDB(*cfg.pool.ConnConfig.Copy())
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close PostgreSQL migration connection: %w", closeErr))
		}
	}()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, goose.WithSessionLocker(locker))
	if errors.Is(err, goose.ErrNoMigrations) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("initialize PostgreSQL migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return 0, fmt.Errorf("apply PostgreSQL migrations: %w", err)
	}
	return len(results), nil
}
