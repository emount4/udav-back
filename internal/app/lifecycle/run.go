package lifecycle

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	corelog "github.com/emount4/udav-back/internal/core/logger"
)

type Server interface {
	Run(ctx context.Context) error
}

func Run(server Server, logger corelog.Logger, logFile io.Closer) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := server.Run(ctx)
	if runErr != nil {
		logger.Error(ctx, "API stopped", corelog.Field{Key: "error", Value: runErr})
	}
	if err := logFile.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close log file:", err)
		os.Exit(1)
	}
	if runErr != nil {
		os.Exit(1)
	}
}
