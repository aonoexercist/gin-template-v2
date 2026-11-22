// Package server owns the HTTP server lifecycle and routing.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gin-template-v2/internal/config"
	"gin-template-v2/internal/platform/dbgen"
)

// sessionCleaner is satisfied by *dbgen.Queries, allowing the server to run
// periodic expired-session cleanup without depending on the full auth package.
type sessionCleaner interface {
	CleanExpiredSessions(ctx context.Context) (int64, error)
}

type Server struct {
	http            *http.Server
	log             *slog.Logger
	shutdownTimeout time.Duration
	cleaner         sessionCleaner
}

func New(cfg config.Config, log *slog.Logger, pool *pgxpool.Pool) *Server {
	q := dbgen.New(pool)
	return &Server{
		http: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
			Handler:           NewRouter(cfg, log, pool),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       cfg.HTTP.ReadTimeout,
			WriteTimeout:      cfg.HTTP.WriteTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
		},
		log:             log,
		shutdownTimeout: cfg.HTTP.ShutdownTimeout,
		cleaner:         q,
	}
}

// Run serves until ctx is cancelled (SIGINT/SIGTERM), then drains in-flight
// requests for up to the shutdown timeout.
func (s *Server) Run(ctx context.Context) error {
	// Start background goroutine to periodically remove expired sessions.
	go s.runSessionCleanup(ctx)

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	s.log.Info("shutting down", "timeout", s.shutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	s.log.Info("server stopped")
	return nil
}

// runSessionCleanup deletes expired sessions every 24 hours until ctx is cancelled.
func (s *Server) runSessionCleanup(ctx context.Context) {
	const interval = 24 * time.Hour

	// Run once immediately at startup so stale rows from a previous run are cleared.
	s.cleanSessions(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanSessions(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) cleanSessions(ctx context.Context) {
	n, err := s.cleaner.CleanExpiredSessions(ctx)
	if err != nil {
		s.log.Error("session cleanup failed", "error", err)
		return
	}
	if n > 0 {
		s.log.Info("expired sessions removed", "count", n)
	}
}
