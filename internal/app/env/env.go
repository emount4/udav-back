package env

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

type Snapshot struct {
	values map[string]string
}

func (s Snapshot) Get(key string) string {
	return s.values[key]
}

func MustRead() Snapshot {
	values, err := readDotenv()
	if err != nil {
		panic(fmt.Errorf("read environment: %w", err))
	}

	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		if runtime.GOOS == "windows" {
			key = strings.ToUpper(key)
		}
		values[key] = value
	}

	return Snapshot{values: values}
}

func readDotenv() (map[string]string, error) {
	file, err := os.Open(".env")
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("open .env: %w", err)
	}
	defer file.Close()

	values, err := godotenv.Parse(file)
	if err != nil {
		// Сообщение парсера может содержать секреты из файла.
		return nil, errors.New("could not parse .env")
	}
	return values, nil
}
