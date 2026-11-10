package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emount4/udav-back/internal/app/http/testsupport"
	"github.com/emount4/udav-back/internal/core/db"
	"github.com/emount4/udav-back/internal/core/requestid"
)

type pingFunc func(context.Context) error

func (fn pingFunc) Ping(ctx context.Context) error {
	return fn(ctx)
}

func TestReadinessUsesDatabaseProbe(t *testing.T) {
	database := pingFunc(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > databaseProbeTimeout {
			t.Fatal("database probe has no bounded deadline")
		}
		return nil
	})
	handler := New(testsupport.Logger{}, database)
	recorder := httptest.NewRecorder()
	handler.Ready(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusNoContent || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected readiness response: %d %v", recorder.Code, recorder.Header())
	}
}

func TestReadinessFailsWhenDatabaseUnavailable(t *testing.T) {
	const detail = "private connection error"
	for _, test := range []struct {
		name     string
		database db.Pinger
	}{
		{"unconfigured", nil},
		{"unreachable", pingFunc(func(context.Context) error { return errors.New(detail) })},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := New(testsupport.Logger{}, test.database)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
			request = request.WithContext(requestid.WithContext(request.Context(), "request-123"))
			handler.Ready(recorder, request)
			body := recorder.Body.String()
			if recorder.Code != http.StatusServiceUnavailable ||
				!strings.Contains(body, "POSTGRES_UNAVAILABLE") || !strings.Contains(body, "request-123") ||
				strings.Contains(body, detail) {
				t.Fatalf("unexpected readiness response: %d %s", recorder.Code, body)
			}

			live := httptest.NewRecorder()
			handler.Live(live, httptest.NewRequest(http.MethodGet, "/health/live", nil))
			if live.Code != http.StatusNoContent {
				t.Fatalf("unexpected liveness status: %d", live.Code)
			}
		})
	}
}
