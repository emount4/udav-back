package s3

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/emount4/udav-back/internal/core/objectstorage"
	"github.com/minio/minio-go/v7"
)

// Ограничение ключа также исключает управляющие символы в подписываемой политике.
var uploadKey = regexp.MustCompile(`^pending/[a-zA-Z0-9_-]{1,128}$`)

func (s *Storage) PresignUpload(ctx context.Context, request objectstorage.UploadRequest) (objectstorage.UploadForm, error) {
	if err := ctx.Err(); err != nil {
		return objectstorage.UploadForm{}, fmt.Errorf("sign S3 upload: %w", err)
	}
	if !uploadKey.MatchString(request.Key) {
		return objectstorage.UploadForm{}, fmt.Errorf("sign S3 upload: invalid pending object key")
	}
	if !request.ExpiresAt.After(time.Now()) {
		return objectstorage.UploadForm{}, fmt.Errorf("sign S3 upload: expiration must be in the future")
	}
	if !validContentType(request.ContentType) {
		return objectstorage.UploadForm{}, fmt.Errorf("sign S3 upload: invalid content type")
	}

	policy := minio.NewPostPolicy()
	if err := policy.SetBucket(s.uploads.bucket); err != nil {
		return objectstorage.UploadForm{}, fmt.Errorf("set S3 upload bucket: %w", err)
	}
	if err := policy.SetKey(request.Key); err != nil {
		return objectstorage.UploadForm{}, fmt.Errorf("set S3 upload key: %w", err)
	}
	if err := policy.SetContentType(request.ContentType); err != nil {
		return objectstorage.UploadForm{}, fmt.Errorf("set S3 upload content type: %w", err)
	}
	if err := policy.SetContentLengthRange(1, request.MaxBytes); err != nil {
		return objectstorage.UploadForm{}, fmt.Errorf("set S3 upload size range: %w", err)
	}
	if err := policy.SetExpires(request.ExpiresAt); err != nil {
		return objectstorage.UploadForm{}, fmt.Errorf("set S3 upload expiration: %w", err)
	}
	url, fields, err := s.signer.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return objectstorage.UploadForm{}, fmt.Errorf("sign S3 upload: %w", err)
	}
	return objectstorage.UploadForm{URL: url.String(), Fields: fields}, nil
}

func (s *Storage) CopyUpload(ctx context.Context, request objectstorage.CopyUploadRequest) error {
	if !uploadKey.MatchString(request.SourceKey) {
		return fmt.Errorf("copy S3 upload: invalid pending object key")
	}
	if request.SourceETag == "" || strings.ContainsAny(request.SourceETag, "\"*, \t\r\n") {
		return fmt.Errorf("copy S3 upload: an exact source ETag is required")
	}
	if !validContentType(request.ContentType) {
		return fmt.Errorf("copy S3 upload: invalid content type")
	}
	_, err := s.assets.client.Client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket:          s.assets.bucket,
		Object:          request.DestinationKey,
		ReplaceMetadata: true,
		ContentType:     request.ContentType,
		ReplaceTags:     true,
	}, minio.CopySrcOptions{
		Bucket:    s.uploads.bucket,
		Object:    request.SourceKey,
		MatchETag: `"` + request.SourceETag + `"`,
		VersionID: url.QueryEscape(request.SourceVersionID),
	})
	if err != nil {
		if minio.ToErrorResponse(err).Code == minio.PreconditionFailed {
			err = objectstorage.ErrObjectChanged
		}
		return fmt.Errorf("copy S3 upload: %w", objectError(err))
	}
	return nil
}

func validContentType(value string) bool {
	parsed, _, err := mime.ParseMediaType(value)
	return err == nil && parsed == value && strings.Contains(value, "/")
}
