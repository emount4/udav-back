package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/emount4/udav-back/internal/app/env"
	"github.com/emount4/udav-back/internal/platform/postgres"
)

func main() {
	settings := env.MustRead()
	config := postgres.MustLoadConfig(settings.Get)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	applied, err := postgres.ApplyMigrations(ctx, config, os.DirFS("migrations"))
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Applied PostgreSQL migrations: %d\n", applied)
}
