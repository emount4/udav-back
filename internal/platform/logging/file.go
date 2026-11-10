package logging

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/emount4/udav-back/internal/core/lifecycle"
	corelog "github.com/emount4/udav-back/internal/core/logger"
)

func MustOpen(cfg Config, console io.Writer) (corelog.Logger, lifecycle.Resource) {
	if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0o750); err != nil {
		panic(fmt.Errorf("create log directory: %w", err))
	}

	file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		panic(fmt.Errorf("open log file: %w", err))
	}
	return New(logOutputs{file: file, console: console}), file
}

type logOutputs struct {
	file    io.Writer
	console io.Writer
}

func (w logOutputs) Write(data []byte) (int, error) {
	// При отказе одного выхода всё равно пытаемся записать во второй.
	fileBytes, fileErr := w.file.Write(data)
	consoleBytes, consoleErr := w.console.Write(data)
	return min(fileBytes, consoleBytes), errors.Join(fileErr, consoleErr)
}
