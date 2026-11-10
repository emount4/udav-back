package db

import "context"

type Transactor interface {
	// WithinTx запускает fn один раз после открытия транзакции. DBTX действителен только внутри fn.
	WithinTx(ctx context.Context, fn func(tx DBTX) error) error
}
