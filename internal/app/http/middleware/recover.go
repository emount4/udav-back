package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/emount4/udav-back/internal/app/http/response"
	corelog "github.com/emount4/udav-back/internal/core/logger"
)

func Recover(logger corelog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracked := trackResponse(w)
			defer func() {
				failure := recover()
				if failure == nil {
					return
				}
				if tracked.conn != nil {
					_ = tracked.conn.Close()
				}
				if failure == http.ErrAbortHandler {
					panic(failure)
				}

				logger.Error(r.Context(), "HTTP handler panicked",
					corelog.Field{Key: "method", Value: r.Method},
					corelog.Field{Key: "route", Value: r.Pattern},
					corelog.Field{Key: "panicType", Value: fmt.Sprintf("%T", failure)},
					corelog.Field{Key: "stack", Value: string(debug.Stack())},
				)

				if tracked.status != 0 || tracked.conn != nil {
					// Ответ уже начался. ErrAbortHandler прерывает его без повторного стека от net/http.
					panic(http.ErrAbortHandler)
				}

				response.WriteInternalError(tracked, r)
			}()
			next.ServeHTTP(tracked, r)
		})
	}
}
