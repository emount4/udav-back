package httpserver

import (
	"context"
	"strings"

	"github.com/emount4/udav-back/internal/core"
)

type errorLogWriter struct {
	logger core.Logger
}

func (w errorLogWriter) Write(message []byte) (int, error) {
	w.logger.Error(context.Background(), strings.TrimSuffix(string(message), "\n"))
	return len(message), nil
}
