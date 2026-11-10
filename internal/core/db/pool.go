package db

import (
	"context"

	"github.com/emount4/udav-back/internal/core/lifecycle"
)

type Pool interface {
	DBTX
	Transactor
	Ping(ctx context.Context) error
	lifecycle.Resource
}
