package objectstorage

import (
	"context"
	"errors"
	"io"
)

var ErrNotFound = errors.New("object not found")

type Metadata struct {
	ContentType string
	Size        int64 // Точный размер объекта в байтах.
	ETag        string
	VersionID   string // Версия объекта, если хранилище её возвращает.
}

type Store interface {
	// Put сохраняет объект из body и заменяет его, если ключ уже существует.
	Put(ctx context.Context, key string, body io.Reader, metadata Metadata) error
	// Get возвращает поток и метаданные. Вызывающий закрывает поток. Для отсутствующего объекта возвращает ErrNotFound.
	Get(ctx context.Context, key string) (io.ReadCloser, Metadata, error)
	// Stat возвращает метаданные. Отсутствие объекта проверяется через errors.Is и ErrNotFound.
	Stat(ctx context.Context, key string) (Metadata, error)
}
