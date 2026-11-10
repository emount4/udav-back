package main

import (
	"os"

	"github.com/emount4/udav-back/internal/app"
	"github.com/emount4/udav-back/internal/app/config"
	"github.com/emount4/udav-back/internal/app/httpserver"
	"github.com/emount4/udav-back/internal/platform/logging"
)

func main() {
	env := config.MustReadEnv()
	httpConfig := config.MustLoadHTTP(env)
	logger := logging.New(os.Stderr)
	server := httpserver.New(httpConfig, logger)
	app.Run(server, logger)
}
