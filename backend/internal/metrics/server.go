// Package metrics provides development-stage Prometheus instrumentation for
// the easy-im backend processes. It exposes a small HTTP server on its own
// port so business routes are never polluted. Every entry point is nil-safe:
// when no metrics server is configured the process behaves exactly as before.
package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Namespace prefixes every easy-im metric.
const Namespace = "easyim"

// Server serves Prometheus metrics on its own address.
type Server struct {
	addr string
	srv  *http.Server
	log  *slog.Logger
}

// NewServer builds an HTTP server that serves Prometheus metrics. A nil addr
// disables the server entirely (callers must skip Start).
func NewServer(addr string, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		addr: addr,
		log:  log,
		srv:  &http.Server{Addr: addr, Handler: promhttp.Handler()},
	}
}

// Addr returns the configured listen address ("" when disabled).
func (s *Server) Addr() string {
	if s == nil {
		return ""
	}
	return s.addr
}

// Start begins serving metrics in a goroutine bound to the server lifetime.
func (s *Server) Start() error {
	if s == nil || s.addr == "" {
		return nil
	}
	go func() {
		s.log.Info("metrics listening", "service", "metrics", "addr", s.addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("metrics listen failed", "service", "metrics", "addr", s.addr, "error", err)
		}
	}()
	return nil
}

// Shutdown gracefully stops the metrics server.
func (s *Server) Shutdown(timeout time.Duration) {
	if s == nil || s.addr == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		s.log.Warn("metrics shutdown", "service", "metrics", "error", err)
	}
}
