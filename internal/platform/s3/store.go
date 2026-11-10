package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/emount4/udav-back/internal/core/objectstorage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type store struct {
	client *minio.Core
	bucket string
}

type Storage struct {
	assets  store
	uploads store
	signer  *minio.Core
}

var (
	_ objectstorage.Store        = (*store)(nil)
	_ objectstorage.UploadSigner = (*Storage)(nil)
	_ objectstorage.UploadCopier = (*Storage)(nil)
)

func Open(cfg Config) (*Storage, error) {
	client, err := newClient(cfg, cfg.endpoint)
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	signer := client
	if cfg.uploadEndpoint != cfg.endpoint {
		signer, err = newClient(cfg, cfg.uploadEndpoint)
		if err != nil {
			return nil, fmt.Errorf("create S3 upload signer: %w", err)
		}
	}
	return &Storage{
		assets:  store{client: client, bucket: cfg.bucket},
		uploads: store{client: client, bucket: cfg.uploadBucket},
		signer:  signer,
	}, nil
}

func (s *Storage) Assets() objectstorage.Store {
	return &s.assets
}

func (s *Storage) Uploads() objectstorage.Store {
	return &s.uploads
}

func newClient(cfg Config, endpoint endpoint) (*minio.Core, error) {
	return minio.NewCore(endpoint.host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.accessKey, cfg.secretKey, ""),
		Secure:       endpoint.secure,
		Region:       cfg.region,
		BucketLookup: minio.BucketLookupPath,
	})
}

func (s *store) Put(ctx context.Context, key string, body io.Reader, metadata objectstorage.Metadata) error {
	if metadata.Size < 0 {
		return fmt.Errorf("put S3 object %q: size must be non-negative", key)
	}
	_, err := s.client.Client.PutObject(ctx, s.bucket, key, body, metadata.Size, minio.PutObjectOptions{
		ContentType: metadata.ContentType,
	})
	if err != nil {
		return fmt.Errorf("put S3 object %q: %w", key, err)
	}
	return nil
}

func (s *store) Get(ctx context.Context, key string) (io.ReadCloser, objectstorage.Metadata, error) {
	// Core.GetObject выполняет запрос сразу, поэтому ошибка видна до чтения потока.
	body, info, _, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, objectstorage.Metadata{}, fmt.Errorf("get S3 object %q: %w", key, objectError(err))
	}
	return body, objectMetadata(info), nil
}

func (s *store) Stat(ctx context.Context, key string) (objectstorage.Metadata, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return objectstorage.Metadata{}, fmt.Errorf("stat S3 object %q: %w", key, objectError(err))
	}
	return objectMetadata(info), nil
}

func objectMetadata(info minio.ObjectInfo) objectstorage.Metadata {
	return objectstorage.Metadata{
		ContentType: info.ContentType,
		Size:        info.Size,
		ETag:        info.ETag,
		VersionID:   info.VersionID,
	}
}

func objectError(err error) error {
	switch minio.ToErrorResponse(err).Code {
	case minio.NoSuchKey, minio.NoSuchVersion:
		return objectstorage.ErrNotFound
	}
	return err
}
