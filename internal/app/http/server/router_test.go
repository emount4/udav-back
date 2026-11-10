package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emount4/udav-back/internal/app/http/middleware"
	"github.com/emount4/udav-back/internal/platform/logging"
)

func TestRouterErrorsUseAPIEnvelope(t *testing.T) {
	handler := New(Config{}, logging.New(io.Discard)).http.Handler
	for _, test := range []struct {
		method string
		path   string
		status int
		code   string
		allow  string
	}{
		{http.MethodGet, "/v1/missing", http.StatusNotFound, "NOT_FOUND", ""},
		{http.MethodPost, "/health/live", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "GET, HEAD"},
	} {
		t.Run(test.code, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != test.status || recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("unexpected status or content type: %d %v", recorder.Code, recorder.Header())
			}
			if allow := recorder.Header().Get("Allow"); allow != test.allow {
				t.Fatalf("Allow = %q, want %q", allow, test.allow)
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
			requestID := recorder.Header().Get(middleware.RequestIDHeader)
			if requestID == "" || body.Error.RequestID != requestID || body.Error.Code != test.code || body.Error.Message == "" {
				t.Fatalf("unexpected error: %+v", body.Error)
			}
		})
	}
}

func TestRouterPreservesMatchedHandlerResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, r.PathValue("id"), http.StatusNotFound)
	})
	recorder := httptest.NewRecorder()
	withJSONRouterErrors(mux, logging.New(io.Discard)).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/player", nil))
	if recorder.Code != http.StatusNotFound || recorder.Body.String() != "player\n" {
		t.Fatalf("matched response was changed: %d %q", recorder.Code, recorder.Body.String())
	}
}
