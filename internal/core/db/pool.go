package db

import (
	"context"

	"github.com/emount4/udav-back/internal/core/lifecycle"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Pool interface {
	DBTX
	Transactor
	Pinger
	lifecycle.Resource
}
