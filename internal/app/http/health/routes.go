package health

import (
	"net/http"

	"github.com/emount4/udav-back/internal/app/http/router"
)

func RegisterRoutes(routes router.Registrar, handler *Handler) {
	routes.Handle(http.MethodGet, "/health/live", http.HandlerFunc(handler.Live))
	routes.Handle(http.MethodGet, "/health/ready", http.HandlerFunc(handler.Ready))
}
