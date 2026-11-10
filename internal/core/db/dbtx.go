package db

import "context"

type DBTX interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) Row
}

type Result interface {
	RowsAffected() int64
}

// Rows нужно закрыть, чтобы соединение вернулось в пул.
type Rows interface {
	Close()
	Next() bool
	Scan(dest ...any) error
	Err() error
}

// Row сообщает об ошибке при вызове Scan. Отсутствие строки проверяется через errors.Is и sql.ErrNoRows.
type Row interface {
	Scan(dest ...any) error
}
