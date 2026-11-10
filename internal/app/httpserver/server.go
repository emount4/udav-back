package httpserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/emount4/udav-back/internal/core"
)

type Server struct {
	http   *http.Server
	logger core.Logger
}

type Config struct {
	Port           int
	AllowedOrigins []string
}

func New(cfg Config, logger core.Logger) *Server {
	mux := http.NewServeMux()
	registerHealthRoutes(mux)

	return &Server{
		http: &http.Server{
			Addr:              ":" + strconv.Itoa(cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ErrorLog:          log.New(errorLogWriter{logger: logger}, "", 0),
		},
		logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.http.Serve(listener)
	}()
	s.logger.Info(ctx, "API listening", core.Field{Key: "address", Value: listener.Addr().String()})

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.http.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP: %w", err)
		}
		<-serveErr
		return nil
	}
}
