package s3

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emount4/udav-back/internal/core/objectstorage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestPresignedUploadUsesPublicEndpointAndExactPolicy(t *testing.T) {
	storage := openTestStorage(t, "http://s3.internal:8333", "https://upload.example.test")
	form, err := storage.PresignUpload(t.Context(), objectstorage.UploadRequest{
		Key: "pending/upload-1", ContentType: "image/png", MaxBytes: 1024, ExpiresAt: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if form.URL != "https://upload.example.test/udav-uploads/" {
		t.Fatalf("unexpected form URL: %s", form.URL)
	}
	policy, err := base64.StdEncoding.DecodeString(form.Fields["policy"])
	if err != nil {
		t.Fatal(err)
	}
	for _, condition := range []string{
		`["eq","$bucket","udav-uploads"]`,
		`["eq","$key","pending/upload-1"]`,
		`["eq","$Content-Type","image/png"]`,
		`["content-length-range", 1, 1024]`,
	} {
		if !strings.Contains(string(policy), condition) {
			t.Errorf("signed policy is missing %s", condition)
		}
	}
}

func TestCopyUploadPinsSourceVersionAndETag(t *testing.T) {
	transport := testTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut || r.URL.Path != "/udav-assets/originals/image.png" {
			t.Errorf("unexpected copy request: %s %s", r.Method, r.URL)
		}
		for name, want := range map[string]string{
			"X-Amz-Copy-Source":          "udav-uploads/pending/upload-1?versionId=version%2B%3D",
			"X-Amz-Copy-Source-If-Match": `"checked-etag"`,
			"X-Amz-Metadata-Directive":   "REPLACE",
			"Content-Type":               "image/png",
		} {
			if got := r.Header.Get(name); got != want {
				t.Errorf("%s = %q, want %q", name, got, want)
			}
		}
		return &http.Response{
			StatusCode: http.StatusPreconditionFailed,
			Status:     "412 Precondition Failed",
			Header:     http.Header{"Content-Type": {"application/xml"}},
			Body:       io.NopCloser(strings.NewReader(`<Error><Code>PreconditionFailed</Code></Error>`)),
			Request:    r,
		}, nil
	})
	client, err := minio.NewCore("s3.internal:8333", &minio.Options{
		Creds:        credentials.NewStaticV4("test-access", "test-secret", ""),
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
		Transport:    transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	storage := openTestStorage(t, "http://s3.internal:8333", "")
	storage.assets.client = client
	err = storage.CopyUpload(t.Context(), objectstorage.CopyUploadRequest{
		SourceKey:       "pending/upload-1",
		SourceETag:      "checked-etag",
		SourceVersionID: "version+=",
		DestinationKey:  "originals/image.png",
		ContentType:     "image/png",
	})
	if !errors.Is(err, objectstorage.ErrObjectChanged) {
		t.Fatalf("copy error = %v, want ErrObjectChanged", err)
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (transport testTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func openTestStorage(t *testing.T, endpoint, uploadEndpoint string) *Storage {
	t.Helper()
	settings := map[string]string{
		"S3_ENDPOINT":        endpoint,
		"S3_UPLOAD_ENDPOINT": uploadEndpoint,
		"S3_BUCKET":          "udav-assets",
		"S3_UPLOAD_BUCKET":   "udav-uploads",
		"S3_ACCESS_KEY":      "udav_local",
		"S3_SECRET_KEY":      "udav_local_dev",
	}
	storage, err := Open(MustLoadConfig(func(key string) string { return settings[key] }))
	if err != nil {
		t.Fatal(err)
	}
	return storage
}
