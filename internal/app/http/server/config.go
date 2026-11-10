package server

import (
	"strconv"
	"strings"
)

type Config struct {
	Port           int
	AllowedOrigins []string
}

func MustLoadConfig(get func(string) string) Config {
	port := 8080
	if rawPort := get("PORT"); rawPort != "" {
		value, err := strconv.Atoi(rawPort)
		if err != nil || value < 1 || value > 65535 {
			panic("load HTTP config: PORT must be an integer from 1 to 65535")
		}
		port = value
	}

	var origins []string
	for _, entry := range strings.Split(get("ALLOWED_ORIGINS"), ",") {
		if origin := strings.TrimSpace(entry); origin != "" {
			origins = append(origins, origin)
		}
	}

	return Config{
		Port:           port,
		AllowedOrigins: origins,
	}
}
