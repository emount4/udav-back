package response_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emount4/udav-back/internal/app/http/response"
	"github.com/emount4/udav-back/internal/app/http/testsupport"
	"github.com/emount4/udav-back/internal/core/apperror"
	corelog "github.com/emount4/udav-back/internal/core/logger"
	"github.com/emount4/udav-back/internal/core/requestid"
)

func TestWriteErrorMapsWrappedApplicationError(t *testing.T) {
	for _, test := range []struct {
		kind   apperror.Kind
		status int
	}{
		{apperror.InvalidInput, http.StatusBadRequest},
		{apperror.Unauthenticated, http.StatusUnauthorized},
		{apperror.Forbidden, http.StatusForbidden},
		{apperror.NotFound, http.StatusNotFound},
		{apperror.Conflict, http.StatusConflict},
		{apperror.RateLimited, http.StatusTooManyRequests},
		{apperror.Unavailable, http.StatusServiceUnavailable},
	} {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			logger := testsupport.Logger{OnError: func(context.Context, []corelog.Field) {
				t.Error("expected error was logged as a server failure")
			}}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPatch, "/v1/me", nil)
			request = request.WithContext(requestid.WithContext(request.Context(), "request-123"))
			expected := apperror.New(test.kind, "EXPECTED_FAILURE", "Публичное сообщение")
			wrapped := fmt.Errorf("service context: %w", expected)

			response.WriteError(recorder, request, logger, wrapped)

			if recorder.Code != test.status {
				t.Fatalf("got status %d, want %d", recorder.Code, test.status)
			}
			var body struct {
				Error struct {
					Code      string `json:"code"`
					Message   string `json:"message"`
					RequestID string `json:"requestId"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != expected.Code() || body.Error.Message != expected.Message() || body.Error.RequestID != "request-123" {
				t.Fatalf("unexpected response: %+v", body.Error)
			}
		})
	}
}

func TestWriteErrorKeepsUnexpectedCauseInLogs(t *testing.T) {
	var loggedError error
	var loggedRequestID string
	logger := testsupport.Logger{OnError: func(ctx context.Context, fields []corelog.Field) {
		loggedRequestID = requestid.FromContext(ctx)
		for _, field := range fields {
			if field.Key == "error" {
				loggedError, _ = field.Value.(error)
			}
		}
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request = request.WithContext(requestid.WithContext(request.Context(), "request-456"))
	cause := errors.New("database connection refused")

	response.WriteError(recorder, request, logger, fmt.Errorf("load profile: %w", cause))

	body := recorder.Body.String()
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(body, "INTERNAL_ERROR") ||
		!strings.Contains(body, "request-456") || strings.Contains(body, cause.Error()) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, body)
	}
	if !errors.Is(loggedError, cause) || loggedRequestID != "request-456" {
		t.Fatalf("cause or request ID missing from the log: %v, %q", loggedError, loggedRequestID)
	}
}
