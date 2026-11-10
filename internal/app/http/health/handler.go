package health

import (
	"context"
	"net/http"
	"time"

	"github.com/emount4/udav-back/internal/app/http/response"
	"github.com/emount4/udav-back/internal/core/apperror"
	"github.com/emount4/udav-back/internal/core/db"
	corelog "github.com/emount4/udav-back/internal/core/logger"
)

const databaseProbeTimeout = time.Second

var errPostgresUnavailable = apperror.New(apperror.Unavailable, "POSTGRES_UNAVAILABLE", "База данных недоступна")

type Handler struct {
	logger   corelog.Logger
	database db.Pinger
}

func New(logger corelog.Logger, database db.Pinger) *Handler {
	return &Handler{logger: logger, database: database}
}

func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.database == nil {
		response.WriteError(w, r, h.logger, errPostgresUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), databaseProbeTimeout)
	defer cancel()
	if err := h.database.Ping(ctx); err != nil {
		h.logger.Warn(r.Context(), "PostgreSQL readiness check failed", corelog.Field{Key: "error", Value: err})
		response.WriteError(w, r, h.logger, errPostgresUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
