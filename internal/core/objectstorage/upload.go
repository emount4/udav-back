package objectstorage

import (
	"context"
	"errors"
	"time"
)

var ErrObjectChanged = errors.New("object changed before copy")

type UploadRequest struct {
	Key         string
	ContentType string
	MaxBytes    int64
	ExpiresAt   time.Time
}

type UploadForm struct {
	URL    string
	Fields map[string]string
}

type UploadSigner interface {
	// PresignUpload подписывает POST-форму для точного ключа во временном хранилище.
	PresignUpload(ctx context.Context, request UploadRequest) (UploadForm, error)
}

type CopyUploadRequest struct {
	SourceKey       string
	SourceETag      string
	SourceVersionID string
	DestinationKey  string
	ContentType     string
}

type UploadCopier interface {
	// CopyUpload сверяет ETag проверенного объекта и при наличии использует его версию.
	CopyUpload(ctx context.Context, request CopyUploadRequest) error
}
