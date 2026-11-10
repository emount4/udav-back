package postgres

import (
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const startupTimeout = 5 * time.Second

type Config struct {
	pool *pgxpool.Config
}

func MustLoadConfig(get func(string) string) Config {
	dsn := strings.TrimSpace(get("DATABASE_URL"))
	if dsn == "" {
		return Config{}
	}

	pool, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// Диагностика парсера может содержать пароль из строки подключения.
		panic("invalid DATABASE_URL")
	}
	if pool.MinConns < 0 || pool.MinConns > pool.MaxConns ||
		pool.MinIdleConns < 0 || pool.MinIdleConns > pool.MaxConns {
		panic("DATABASE_URL: pool_min_conns and pool_min_idle_conns must be between 0 and pool_max_conns")
	}
	if pool.HealthCheckPeriod <= 0 {
		panic("DATABASE_URL: pool_health_check_period must be positive")
	}

	// pgx может продолжить открывать соединение после отмены ожидания пула.
	if timeout := pool.ConnConfig.ConnectTimeout; timeout <= 0 || timeout > startupTimeout {
		pool.ConnConfig.ConnectTimeout = startupTimeout
	}
	return Config{pool: pool}
}

func (c Config) Enabled() bool {
	return c.pool != nil
}
