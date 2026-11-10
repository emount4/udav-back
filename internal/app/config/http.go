package config

import (
	"strconv"
	"strings"

	"github.com/emount4/udav-back/internal/app/httpserver"
)

func MustLoadHTTP(env Env) httpserver.Config {
	port := 8080
	if rawPort := env.values["PORT"]; rawPort != "" {
		value, err := strconv.Atoi(rawPort)
		if err != nil || value < 1 || value > 65535 {
			panic("load HTTP config: PORT must be an integer from 1 to 65535")
		}
		port = value
	}

	var origins []string
	for _, entry := range strings.Split(env.values["ALLOWED_ORIGINS"], ",") {
		if origin := strings.TrimSpace(entry); origin != "" {
			origins = append(origins, origin)
		}
	}

	return httpserver.Config{
		Port:           port,
		AllowedOrigins: origins,
	}
}
