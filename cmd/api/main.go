package main

import (
	"context"
	"os"

	"github.com/emount4/udav-back/internal/app/env"
	"github.com/emount4/udav-back/internal/app/http/health"
	"github.com/emount4/udav-back/internal/app/http/router"
	httpserver "github.com/emount4/udav-back/internal/app/http/server"
	"github.com/emount4/udav-back/internal/app/lifecycle"
	"github.com/emount4/udav-back/internal/core/db"
	corelife "github.com/emount4/udav-back/internal/core/lifecycle"
	"github.com/emount4/udav-back/internal/platform/logging"
	"github.com/emount4/udav-back/internal/platform/postgres"
)

func main() {
	settings := env.MustRead()

	logConfig := logging.LoadConfig(settings.Get)
	logger, logFile := logging.MustOpen(logConfig, os.Stderr)

	httpConfig := httpserver.MustLoadConfig(settings.Get)

	resources := []corelife.Resource{logFile}
	var database db.Pinger
	if dbConfig := postgres.MustLoadConfig(settings.Get); dbConfig.Enabled() {
		pool := postgres.MustOpen(context.Background(), dbConfig)
		resources = append(resources, pool)
		database = pool
	}

	probes := health.New(logger, database)
	routes := router.New(logger)
	health.RegisterRoutes(routes, probes)
	server := httpserver.New(httpConfig, logger, routes)
	lifecycle.Run(server, logger, resources...)
}
