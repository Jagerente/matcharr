package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/transport/http/httperr"
)

// Observer receives what the HTTP layer sees. It is optional: an instance
// without metrics wired passes nil and pays nothing.
type Observer interface {
	RequestStarted()
	RequestDone()
	ObserveRequest(route, method string, status int, d time.Duration)
	ObserveError(reason string)
}

type ObserveConfig struct {
	Logger   *slog.Logger
	Observer Observer
	// Skipper exempts routes that must not enter the request timings, the
	// event socket lives for a whole session.
	Skipper func(c echo.Context) bool
	// TrimPrefix strips the URL base from route labels so metrics stay
	// stable regardless of how the instance is mounted.
	TrimPrefix string
}

// Observe logs every request and feeds the metrics. It resolves errors
// through the central handler itself, so the status and reason it records
// are the ones the client actually received.
func Observe(cfg ObserveConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}
			if cfg.Observer != nil {
				cfg.Observer.RequestStarted()
				defer cfg.Observer.RequestDone()
			}

			start := time.Now()
			err := next(c)
			if err != nil {
				c.Error(err)
			}
			duration := time.Since(start)
			status := c.Response().Status

			cfg.Logger.Debug("http",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", status,
				"duration", duration.Round(time.Millisecond),
			)
			if cfg.Observer != nil {
				cfg.Observer.ObserveRequest(routeLabel(c.Path(), cfg.TrimPrefix), c.Request().Method, status, duration)
				if err != nil {
					_, reason, _ := httperr.Map(err)
					cfg.Observer.ObserveError(string(reason))
				}
			}
			return nil
		}
	}
}

// routeLabel converts echo's ":param" pattern into "{param}" and keeps the
// catch-all as "/", matching the labels the dashboards were built on.
func routeLabel(path, trimPrefix string) string {
	if trimPrefix != "" {
		path = strings.TrimPrefix(path, trimPrefix)
	}
	if path == "" || path == "/*" {
		return "/"
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if name, ok := strings.CutPrefix(segment, ":"); ok {
			segments[i] = "{" + name + "}"
		}
	}
	return strings.Join(segments, "/")
}
