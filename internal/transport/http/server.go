package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const shutdownTimeout = 5 * time.Second

type Server struct {
	logger *slog.Logger
	echo   *echo.Echo
	addr   string
}

func New(cfg Config, d Deps) *Server {
	e := newRouter(cfg, d)
	e.Server.ReadTimeout = cfg.ReadTimeout
	e.Server.ReadHeaderTimeout = cfg.ReadHeaderTimeout
	e.Server.IdleTimeout = cfg.IdleTimeout

	return &Server{
		logger: d.Logger,
		echo:   e,
		addr:   fmt.Sprintf(":%d", cfg.Port),
	}
}

func (s *Server) Handler() http.Handler { return s.echo }

func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.echo.Listener = ln

	errCh := make(chan error, 1)
	go func() {
		if err := s.echo.Start(""); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	s.logger.Info("server started", "addr", s.addr)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return s.echo.Shutdown(shutdownCtx)
	}
}
