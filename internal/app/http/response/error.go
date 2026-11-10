package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/emount4/udav-back/internal/core/apperror"
	corelog "github.com/emount4/udav-back/internal/core/logger"
	"github.com/emount4/udav-back/internal/core/requestid"
)

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func WriteError(w http.ResponseWriter, r *http.Request, logger corelog.Logger, err error) {
	var expected *apperror.Error
	if errors.As(err, &expected) && expected != nil {
		if status := statusForKind(expected.Kind()); status != 0 {
			writeError(w, r, status, expected.Code(), expected.Message())
			return
		}
	}

	logger.Error(r.Context(), "HTTP request failed",
		corelog.Field{Key: "method", Value: r.Method},
		corelog.Field{Key: "route", Value: r.Pattern},
		corelog.Field{Key: "error", Value: err},
	)
	WriteInternalError(w, r)
}

func WriteInternalError(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
}

func WriteMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Метод не поддерживается")
}

func statusForKind(kind apperror.Kind) int {
	switch kind {
	case apperror.InvalidInput:
		return http.StatusBadRequest
	case apperror.Unauthenticated:
		return http.StatusUnauthorized
	case apperror.Forbidden:
		return http.StatusForbidden
	case apperror.NotFound:
		return http.StatusNotFound
	case apperror.Conflict:
		return http.StatusConflict
	case apperror.RateLimited:
		return http.StatusTooManyRequests
	case apperror.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return 0
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	header := w.Header()
	header.Del("Content-Length")
	header.Del("Content-Encoding")
	header.Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// После отправки заголовков повторный ответ при ошибке записи невозможен.
	_ = json.NewEncoder(w).Encode(struct {
		Error apiError `json:"error"`
	}{
		Error: apiError{
			Code:      code,
			Message:   message,
			RequestID: requestid.FromContext(r.Context()),
		},
	})
}
