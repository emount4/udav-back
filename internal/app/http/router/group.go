package router

import (
	"net/http"
	"slices"
	"strings"

	"github.com/emount4/udav-back/internal/app/http/middleware"
)

type Registrar interface {
	Handle(method, path string, handler http.Handler, middlewares ...middleware.Middleware)
}

type Group struct {
	mux         *http.ServeMux
	prefix      string
	middlewares []middleware.Middleware
}

func (rt *Router) Handle(method, path string, handler http.Handler, middlewares ...middleware.Middleware) {
	Group{mux: rt.mux}.Handle(method, path, handler, middlewares...)
}

func (rt *Router) Group(prefix string, middlewares ...middleware.Middleware) Group {
	return Group{mux: rt.mux}.Group(prefix, middlewares...)
}

// Group наследует префикс и middleware родителя. Пустой префикс добавляет только middleware.
func (g Group) Group(prefix string, middlewares ...middleware.Middleware) Group {
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		panic("router group prefix must start with /")
	}
	return Group{
		mux:         g.mux,
		prefix:      g.prefix + strings.TrimRight(prefix, "/"),
		middlewares: slices.Concat(g.middlewares, middlewares),
	}
}

func (g Group) Handle(method, path string, handler http.Handler, middlewares ...middleware.Middleware) {
	if method == "" {
		panic("router method is required")
	}
	if path != "" && !strings.HasPrefix(path, "/") {
		panic("router path must start with /")
	}
	pattern := method + " " + g.prefix + path
	stack := slices.Concat(g.middlewares, middlewares)
	g.mux.Handle(pattern, middleware.Chain(handler, stack...))
}
