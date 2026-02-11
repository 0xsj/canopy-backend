package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Server wraps net/http.Server with configuration and graceful shutdown.
type Server struct {
	http *http.Server
	log  logger.Logger
	cfg  Config
}

// New creates an HTTP server. The handler is typically a ServeMux
// wrapped with middleware via Chain.
func New(cfg Config, handler http.Handler, log logger.Logger) *Server {
	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		log: log,
		cfg: cfg,
	}
}

// Start begins listening and serving. Blocks until the server stops.
// Returns http.ErrServerClosed on graceful shutdown.
func (s *Server) Start() error {
	s.log.Info("http server starting",
		logger.String("addr", s.http.Addr),
	)

	ln, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return fmt.Errorf("httpserver: listen: %w", err)
	}

	s.log.Info("http server listening",
		logger.String("addr", ln.Addr().String()),
	)

	return s.http.Serve(ln)
}

// Shutdown gracefully stops the server, waiting for in-flight requests
// up to the configured shutdown timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")

	ctx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("httpserver: shutdown: %w", err)
	}

	s.log.Info("http server stopped")
	return nil
}

// Addr returns the configured listen address.
func (s *Server) Addr() string {
	return s.http.Addr
}
