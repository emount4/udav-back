package logging

import (
	"context"
	"io"
	"log/slog"

	"github.com/emount4/udav-back/internal/core"
)

type logger struct {
	base *slog.Logger
}

func New(output io.Writer) core.Logger {
	return &logger{base: slog.New(slog.NewJSONHandler(output, nil))}
}

func (l *logger) Debug(ctx context.Context, message string, fields ...core.Field) {
	l.write(ctx, slog.LevelDebug, message, fields)
}

func (l *logger) Info(ctx context.Context, message string, fields ...core.Field) {
	l.write(ctx, slog.LevelInfo, message, fields)
}

func (l *logger) Warn(ctx context.Context, message string, fields ...core.Field) {
	l.write(ctx, slog.LevelWarn, message, fields)
}

func (l *logger) Error(ctx context.Context, message string, fields ...core.Field) {
	l.write(ctx, slog.LevelError, message, fields)
}

func (l *logger) write(ctx context.Context, level slog.Level, message string, fields []core.Field) {
	attrs := make([]slog.Attr, 0, len(fields))
	for _, field := range fields {
		attrs = append(attrs, slog.Any(field.Key, field.Value))
	}
	l.base.LogAttrs(ctx, level, message, attrs...)
}
