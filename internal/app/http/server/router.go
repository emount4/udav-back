package server

import (
	"net/http"

	"github.com/emount4/udav-back/internal/app/http/response"
	"github.com/emount4/udav-back/internal/core/apperror"
	corelog "github.com/emount4/udav-back/internal/core/logger"
)

var errRouteNotFound = apperror.New(apperror.NotFound, "NOT_FOUND", "Маршрут не найден")

func withJSONRouterErrors(mux *http.ServeMux, logger corelog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallback, pattern := mux.Handler(r)
		if pattern != "" || r.RequestURI == "*" {
			// ServeHTTP заполняет шаблон маршрута и параметры для PathValue.
			mux.ServeHTTP(w, r)
			return
		}

		fallback.ServeHTTP(&routerErrorWriter{ResponseWriter: w, request: r, logger: logger}, r)
	})
}

type routerErrorWriter struct {
	http.ResponseWriter
	request  *http.Request
	logger   corelog.Logger
	replaced bool
}

func (w *routerErrorWriter) WriteHeader(status int) {
	switch status {
	case http.StatusNotFound:
		response.WriteError(w.ResponseWriter, w.request, w.logger, errRouteNotFound)
	case http.StatusMethodNotAllowed:
		response.WriteMethodNotAllowed(w.ResponseWriter, w.request)
	default:
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.replaced = true
}

func (w *routerErrorWriter) Write(data []byte) (int, error) {
	if w.replaced {
		return len(data), nil
	}
	return w.ResponseWriter.Write(data)
}
