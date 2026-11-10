package logging

type Config struct {
	FilePath string
}

func LoadConfig(get func(string) string) Config {
	path := get("LOG_FILE")
	if path == "" {
		path = "logs/api.log"
	}
	return Config{FilePath: path}
}
