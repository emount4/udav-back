package server

import (
	"context"
	"strings"

	corelog "github.com/emount4/udav-back/internal/core/logger"
)

type errorLogWriter struct {
	logger corelog.Logger
}

func (w errorLogWriter) Write(message []byte) (int, error) {
	w.logger.Error(context.Background(), strings.TrimSuffix(string(message), "\n"))
	return len(message), nil
}
