package middleware

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/emount4/udav-back/internal/core/requestid"
)

const RequestIDHeader = "X-Request-ID"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id [16]byte
		rand.Read(id[:])
		id[6] = id[6]&0x0f | 0x40
		id[8] = id[8]&0x3f | 0x80
		requestID := fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])

		w.Header().Set(RequestIDHeader, requestID)
		ctx := requestid.WithContext(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
