package main

import (
	"os"

	"github.com/emount4/udav-back/internal/app/env"
	httpserver "github.com/emount4/udav-back/internal/app/http/server"
	"github.com/emount4/udav-back/internal/app/lifecycle"
	"github.com/emount4/udav-back/internal/platform/logging"
)

func main() {
	settings := env.MustRead()

	logConfig := logging.LoadConfig(settings.Get)
	logger, logFile := logging.MustOpen(logConfig, os.Stderr)

	httpConfig := httpserver.MustLoadConfig(settings.Get)
	server := httpserver.New(httpConfig, logger)
	lifecycle.Run(server, logger, logFile)
}
