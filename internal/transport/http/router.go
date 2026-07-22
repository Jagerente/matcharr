package http

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/transport/http/api"
	"github.com/Jagerente/matcharr/internal/transport/http/binder"
	"github.com/Jagerente/matcharr/internal/transport/http/httperr"
	"github.com/Jagerente/matcharr/internal/transport/http/middleware"
	"github.com/Jagerente/matcharr/internal/transport/http/validator"
	"github.com/Jagerente/matcharr/internal/transport/http/ws"
)

type Config struct {
	Port              int
	URLBase           string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	BodyLimit         int64
}

type Limiters struct {
	Create   middleware.RateLimiter
	Join     middleware.RateLimiter
	Action   middleware.RateLimiter
	Catalog  middleware.RateLimiter
	Image    middleware.RateLimiter
	ClientIP middleware.ClientIP
}

type Deps struct {
	Logger          *slog.Logger
	Sessions        api.SessionService
	Catalog         api.CatalogService
	Requests        api.RequestService
	Posters         api.PosterStore
	Hub             *ws.Hub
	Static          fs.FS
	RequestsEnabled bool
	Limiters        Limiters
	Observer        middleware.Observer
	PosterObserver  api.PosterObserver
}

func newRouter(cfg Config, d Deps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Binder = binder.New()
	e.Validator = validator.New()
	e.HTTPErrorHandler = httperr.Handle

	// /healthz stays at the root regardless of URLBase so the compose
	// healthcheck does not depend on the proxy setup.
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	base := cfg.URLBase
	wsPath := base + "/api/sessions/:code/ws"
	app := e.Group(base, middleware.Observe(middleware.ObserveConfig{
		Logger:     d.Logger,
		Observer:   d.Observer,
		TrimPrefix: base,
		Skipper: func(c echo.Context) bool {
			return c.Path() == wsPath
		},
	}))

	registerAPI(app, cfg, d)
	// Echo's catch-all does not match the bare group root, so the SPA
	// registers both.
	spa := spaHandler(d.Static)
	app.GET("/", spa)
	app.GET("/*", spa)
	registerBaseRedirects(e, base)

	return e
}

func registerAPI(app *echo.Group, cfg Config, d Deps) {
	sessions := api.NewSessionHandler(d.Sessions)
	catalog := api.NewCatalogHandler(d.Catalog, d.RequestsEnabled)
	requests := api.NewRequestHandler(d.Requests)
	posters := api.NewPosterHandler(d.Posters, d.PosterObserver)

	lim := d.Limiters
	g := app.Group("/api", middleware.BodyLimit(cfg.BodyLimit))

	// Without this an unknown /api path would fall through to the SPA and
	// a client would get index.html with a 200 where it expects JSON.
	g.RouteNotFound("/*", func(echo.Context) error {
		return apperr.New(apperr.KindNotFound, apperr.ReasonBadRequest, "unknown endpoint")
	})

	g.GET("/config", catalog.Config, guard(lim.Catalog, lim.ClientIP)...)
	g.GET("/genres", catalog.Genres, guard(lim.Catalog, lim.ClientIP)...)
	g.GET("/presets", catalog.Presets, guard(lim.Catalog, lim.ClientIP)...)
	g.GET("/image/:size/:file", posters.Serve, guard(lim.Image, lim.ClientIP)...)

	action := guard(lim.Action, lim.ClientIP)
	g.POST("/sessions", sessions.Create, guard(lim.Create, lim.ClientIP)...)
	g.POST("/sessions/:code/join", sessions.Join, guard(lim.Join, lim.ClientIP)...)
	g.GET("/sessions/:code", sessions.State, action...)
	g.GET("/sessions/:code/deck", sessions.Deck, action...)
	g.GET("/sessions/:code/movies/:id", catalog.MovieDetails, guard(lim.Catalog, lim.ClientIP)...)
	g.POST("/sessions/:code/start", sessions.Start, action...)
	g.POST("/sessions/:code/swipe", sessions.Swipe, action...)
	g.POST("/sessions/:code/undo", sessions.Undo, action...)
	g.POST("/sessions/:code/finish", sessions.Finish, action...)
	g.POST("/sessions/:code/leave", sessions.Leave, action...)
	g.POST("/sessions/:code/continue/propose", sessions.ProposeContinue, action...)
	g.POST("/sessions/:code/continue/vote", sessions.VoteContinue, action...)
	g.POST("/sessions/:code/continue/cancel", sessions.CancelContinue, action...)
	g.DELETE("/sessions/:code/participants/:id", sessions.Kick, action...)
	g.GET("/sessions/:code/request/profiles", requests.Profiles, action...)
	g.POST("/sessions/:code/request", requests.Submit, action...)
	g.GET("/sessions/:code/ws", d.Hub.Handle, action...)
}

func guard(limiter middleware.RateLimiter, ip middleware.ClientIP) []echo.MiddlewareFunc {
	if limiter == nil {
		return nil
	}
	return []echo.MiddlewareFunc{middleware.RateLimit(limiter, ip)}
}

func registerBaseRedirects(e *echo.Echo, base string) {
	if base == "" {
		return
	}
	e.GET(base, func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, base+"/")
	})
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, base+"/")
	})
}
