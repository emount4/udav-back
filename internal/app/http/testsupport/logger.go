package testsupport

import (
	"context"

	corelog "github.com/emount4/udav-back/internal/core/logger"
)

type Logger struct {
	OnError func(context.Context, []corelog.Field)
}

func (Logger) Debug(context.Context, string, ...corelog.Field) {}
func (Logger) Info(context.Context, string, ...corelog.Field)  {}
func (Logger) Warn(context.Context, string, ...corelog.Field)  {}

func (l Logger) Error(ctx context.Context, _ string, fields ...corelog.Field) {
	if l.OnError != nil {
		l.OnError(ctx, fields)
	}
}
