package s3

import (
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7/pkg/s3utils"
)

type Config struct {
	endpoint       endpoint
	uploadEndpoint endpoint
	bucket         string
	uploadBucket   string
	region         string
	accessKey      string
	secretKey      string
}

type endpoint struct {
	host   string
	secure bool
}

func MustLoadConfig(get func(string) string) Config {
	internalEndpoint := mustParseEndpoint(get("S3_ENDPOINT"), "S3_ENDPOINT")
	publicEndpoint := internalEndpoint
	if raw := strings.TrimSpace(get("S3_UPLOAD_ENDPOINT")); raw != "" {
		publicEndpoint = mustParseEndpoint(raw, "S3_UPLOAD_ENDPOINT")
	}
	bucket := strings.TrimSpace(get("S3_BUCKET"))
	if err := s3utils.CheckValidBucketNameStrict(bucket); err != nil {
		panic("S3_BUCKET must be a valid bucket name")
	}
	uploadBucket := strings.TrimSpace(get("S3_UPLOAD_BUCKET"))
	if err := s3utils.CheckValidBucketNameStrict(uploadBucket); err != nil || uploadBucket == bucket {
		panic("S3_UPLOAD_BUCKET must be a valid bucket name distinct from S3_BUCKET")
	}
	accessKey := get("S3_ACCESS_KEY")
	secretKey := get("S3_SECRET_KEY")
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		panic("S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}
	region := strings.TrimSpace(get("S3_REGION"))
	if region == "" {
		region = "us-east-1"
	}
	return Config{
		endpoint:       internalEndpoint,
		uploadEndpoint: publicEndpoint,
		bucket:         bucket,
		uploadBucket:   uploadBucket,
		region:         region,
		accessKey:      accessKey,
		secretKey:      secretKey,
	}
}

func mustParseEndpoint(raw, variable string) endpoint {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Hostname() == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.ForceQuery ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		panic(variable + " must be an HTTP or HTTPS URL without credentials, path, query or fragment")
	}
	return endpoint{host: parsed.Host, secure: parsed.Scheme == "https"}
}
