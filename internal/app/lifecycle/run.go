package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"syscall"

	corelife "github.com/emount4/udav-back/internal/core/lifecycle"
	corelog "github.com/emount4/udav-back/internal/core/logger"
)

type Server interface {
	Run(ctx context.Context) error
}

func Run(server Server, logger corelog.Logger, resources ...corelife.Resource) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	failed := false
	if err := server.Run(ctx); err != nil {
		logger.Error(ctx, "API stopped", corelog.Field{Key: "error", Value: err})
		failed = true
	}
	// Закрываем ресурсы в обратном порядке, чтобы логгер оставался доступным до конца остановки.
	for _, resource := range slices.Backward(resources) {
		if err := resource.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close resource:", err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}
