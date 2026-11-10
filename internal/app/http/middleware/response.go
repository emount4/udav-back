package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	conn   net.Conn
}

func trackResponse(w http.ResponseWriter) *responseWriter {
	if tracked, ok := w.(*responseWriter); ok {
		return tracked
	}
	return &responseWriter{ResponseWriter: w}
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	if w.status == 0 && (status >= 200 || status == http.StatusSwitchingProtocols) {
		w.status = status
	}
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) Flush() {
	_ = w.FlushError()
}

func (w *responseWriter) FlushError() error {
	err := http.NewResponseController(w.ResponseWriter).Flush()
	if w.status == 0 && !errors.Is(err, http.ErrNotSupported) {
		w.status = http.StatusOK
	}
	return err
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, stream, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err == nil {
		w.conn = conn
	}
	return conn, stream, err
}
