package middleware

import (
	"net/http"
	"time"

	corelog "github.com/emount4/udav-back/internal/core/logger"
)

func Trace(logger corelog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := trackResponse(w)
			start := time.Now()
			completed := false
			defer func() {
				// Шаблон маршрута позволяет опустить параметры пути и WebSocket-билеты из URL.
				fields := []corelog.Field{
					{Key: "method", Value: r.Method},
					{Key: "route", Value: r.Pattern},
					{Key: "durationMs", Value: time.Since(start).Milliseconds()},
				}

				status := response.status
				if status == 0 && completed && response.conn == nil {
					status = http.StatusOK
				}
				if status != 0 {
					fields = append(fields, corelog.Field{Key: "status", Value: status})
				}
				if response.conn != nil {
					fields = append(fields, corelog.Field{Key: "hijacked", Value: true})
				}
				if !completed {
					fields = append(fields, corelog.Field{Key: "aborted", Value: true})
				}

				logger.Info(r.Context(), "HTTP request", fields...)
			}()

			next.ServeHTTP(response, r)
			completed = true
		})
	}
}
