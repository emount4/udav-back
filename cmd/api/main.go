package main

import (
	"context"
	"os"

	"github.com/emount4/udav-back/internal/app/env"
	httpserver "github.com/emount4/udav-back/internal/app/http/server"
	"github.com/emount4/udav-back/internal/app/lifecycle"
	corelife "github.com/emount4/udav-back/internal/core/lifecycle"
	"github.com/emount4/udav-back/internal/platform/logging"
	"github.com/emount4/udav-back/internal/platform/postgres"
)

func main() {
	settings := env.MustRead()

	logConfig := logging.LoadConfig(settings.Get)
	logger, logFile := logging.MustOpen(logConfig, os.Stderr)

	httpConfig := httpserver.MustLoadConfig(settings.Get)
	server := httpserver.New(httpConfig, logger)

	resources := []corelife.Resource{logFile}
	if dbConfig := postgres.MustLoadConfig(settings.Get); dbConfig.Enabled() {
		pool := postgres.MustOpen(context.Background(), dbConfig)
		resources = append(resources, pool)
	}

	lifecycle.Run(server, logger, resources...)
}
