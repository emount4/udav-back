package response_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emount4/udav-back/internal/app/http/response"
	"github.com/emount4/udav-back/internal/core/apperror"
	"github.com/emount4/udav-back/internal/core/requestid"
	"github.com/emount4/udav-back/internal/platform/logging"
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
	} {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			var output bytes.Buffer
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPatch, "/v1/me", nil)
			request = request.WithContext(requestid.WithContext(request.Context(), "request-123"))
			expected := apperror.New(test.kind, "EXPECTED_FAILURE", "Публичное сообщение")
			wrapped := fmt.Errorf("service context: %w", expected)

			response.WriteError(recorder, request, logging.New(&output), wrapped)

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
			if output.Len() != 0 {
				t.Fatalf("expected error was logged as a server failure: %s", output.String())
			}
		})
	}
}

func TestWriteErrorKeepsUnexpectedCauseInLogs(t *testing.T) {
	var output bytes.Buffer
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request = request.WithContext(requestid.WithContext(request.Context(), "request-456"))
	cause := errors.New("database connection refused")

	response.WriteError(recorder, request, logging.New(&output), fmt.Errorf("load profile: %w", cause))

	body := recorder.Body.String()
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(body, "INTERNAL_ERROR") ||
		!strings.Contains(body, "request-456") || strings.Contains(body, cause.Error()) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, body)
	}
	if !strings.Contains(output.String(), cause.Error()) || !strings.Contains(output.String(), "request-456") {
		t.Fatalf("cause or request ID missing from the log: %s", output.String())
	}
}
