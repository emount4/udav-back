package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type failedOutput struct{}

func (failedOutput) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestFileLoggerWritesToEachAvailableDestination(t *testing.T) {
	t.Run("file fails", func(t *testing.T) {
		var console bytes.Buffer
		path := filepath.Join(t.TempDir(), "api.log")
		logger, file := MustOpen(Config{FilePath: path}, &console)
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		logger.Error(t.Context(), "log entry")
		assertLogEntry(t, console.Bytes())
	})

	t.Run("console fails", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "api.log")
		logger, file := MustOpen(Config{FilePath: path}, failedOutput{})
		logger.Error(t.Context(), "log entry")
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		assertLogEntry(t, data)
	})
}

func assertLogEntry(t *testing.T, data []byte) {
	t.Helper()
	var entry struct {
		Message string `json:"msg"`
	}
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Message != "log entry" {
		t.Fatalf("unexpected log entry: %q", data)
	}
}
