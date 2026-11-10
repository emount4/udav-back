package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/emount4/udav-back/internal/core"
)

type Server interface {
	Run(ctx context.Context) error
}

func Run(server Server, logger core.Logger) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		logger.Error(ctx, "API stopped", core.Field{Key: "error", Value: err})
		os.Exit(1)
	}
}
