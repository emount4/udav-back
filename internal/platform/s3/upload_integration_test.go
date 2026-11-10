package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/emount4/udav-back/internal/core/objectstorage"
	"github.com/minio/minio-go/v7"
)

func TestDirectUploadAndConditionalCopy(t *testing.T) {
	address := os.Getenv("UDAV_S3_TEST_ENDPOINT")
	if address == "" {
		t.Skip("set UDAV_S3_TEST_ENDPOINT to run against the local Compose S3 service")
	}
	storage := openTestStorage(t, address, "")
	id := rand.Text()
	sourceKey, destinationKey := "pending/"+id, "originals/smoke-"+id+".png"
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, target := range []struct {
			bucket string
			key    string
		}{{storage.uploads.bucket, sourceKey}, {storage.assets.bucket, destinationKey}} {
			if err := storage.assets.client.RemoveObject(ctx, target.bucket, target.key, minio.RemoveObjectOptions{}); err != nil {
				t.Error(err)
			}
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	form, err := storage.PresignUpload(ctx, objectstorage.UploadRequest{
		Key: sourceKey, ContentType: "image/jpeg", MaxBytes: 1024, ExpiresAt: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	postUpload(t, ctx, form, sourceKey+"-wrong", []byte("wrong key"), false)
	postUpload(t, ctx, form, sourceKey, make([]byte, 2048), false)
	wrongType := objectstorage.UploadForm{URL: form.URL, Fields: maps.Clone(form.Fields)}
	wrongType.Fields["Content-Type"] = "image/png"
	postUpload(t, ctx, wrongType, sourceKey, []byte("wrong type"), false)
	postUpload(t, ctx, form, sourceKey, []byte("first"), true)
	_, first := readObject(t, ctx, storage.Uploads(), sourceKey)
	if first.ETag == "" {
		t.Fatal("temporary object is missing its ETag")
	}

	content := []byte("replacement")
	postUpload(t, ctx, form, sourceKey, content, true)
	copyRequest := objectstorage.CopyUploadRequest{
		SourceKey: sourceKey, SourceETag: first.ETag, DestinationKey: destinationKey, ContentType: "image/png",
	}
	if err := storage.CopyUpload(ctx, copyRequest); !errors.Is(err, objectstorage.ErrObjectChanged) {
		t.Fatalf("stale ETag: got %v, want ErrObjectChanged", err)
	}
	if _, err := storage.Assets().Stat(ctx, destinationKey); !errors.Is(err, objectstorage.ErrNotFound) {
		t.Fatalf("destination after rejected copy: got %v, want ErrNotFound", err)
	}
	_, latest := readObject(t, ctx, storage.Uploads(), sourceKey)
	copyRequest.SourceETag = latest.ETag
	copyRequest.SourceVersionID = latest.VersionID
	if err := storage.CopyUpload(ctx, copyRequest); err != nil {
		t.Fatal(err)
	}
	data, metadata := readObject(t, ctx, storage.Assets(), destinationKey)
	if !bytes.Equal(data, content) || metadata.Size != int64(len(content)) || metadata.ContentType != "image/png" {
		t.Fatalf("copied object: bytes=%q metadata=%+v", data, metadata)
	}
}

func postUpload(t *testing.T, ctx context.Context, form objectstorage.UploadForm, key string, content []byte, wantSuccess bool) {
	t.Helper()
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	for field, value := range form.Fields {
		if field == "key" {
			value = key
		}
		if err := writer.WriteField(field, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, form.URL, &payload)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if (wantSuccess && response.StatusCode/100 != 2) || (!wantSuccess && response.StatusCode/100 != 4) {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		t.Fatalf("upload status %d, expected success=%v: %s", response.StatusCode, wantSuccess, body)
	}
}

func readObject(t *testing.T, ctx context.Context, store objectstorage.Store, key string) ([]byte, objectstorage.Metadata) {
	t.Helper()
	body, metadata, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		t.Fatal(err)
	}
	return data, metadata
}
