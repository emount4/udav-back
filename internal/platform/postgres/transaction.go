package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emount4/udav-back/internal/core/db"
	"github.com/jackc/pgx/v5"
)

const rollbackTimeout = 5 * time.Second

func (p *pgxPool) WithinTx(ctx context.Context, fn func(db.DBTX) error) (err error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin postgres transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		// Откат должен вернуть соединение в пул даже после отмены запроса.
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
		defer cancel()
		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback postgres transaction: %w", rollbackErr))
		}
	}()

	if err := fn(pgxQueries{executor: tx}); err != nil {
		return fmt.Errorf("execute postgres transaction: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("postgres transaction interrupted before commit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit postgres transaction: %w", err)
	}
	committed = true
	return nil
}
