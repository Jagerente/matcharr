package app

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/Jagerente/matcharr/internal/client/seerr"
	"github.com/Jagerente/matcharr/internal/client/tmdb"
	"github.com/Jagerente/matcharr/internal/config"
	"github.com/Jagerente/matcharr/internal/metrics"
	"github.com/Jagerente/matcharr/internal/poster"
	"github.com/Jagerente/matcharr/internal/preset"
	"github.com/Jagerente/matcharr/internal/service/catalog"
	"github.com/Jagerente/matcharr/internal/service/deck"
	"github.com/Jagerente/matcharr/internal/service/request"
	"github.com/Jagerente/matcharr/internal/service/session"
	transport "github.com/Jagerente/matcharr/internal/transport/http"
	"github.com/Jagerente/matcharr/internal/transport/http/ws"
	"github.com/Jagerente/matcharr/web"
)

type App struct {
	cfg      *config.Config
	logger   *slog.Logger
	sessions *session.Service
	posters  *poster.Store
	server   *transport.Server
	meters   *metrics.Metrics
}

func New(cfg *config.Config, logger *slog.Logger) (*App, error) {
	var meters *metrics.Metrics
	if cfg.Metrics.On() {
		meters = metrics.New()
	}

	tmdbClient := tmdb.NewClient(tmdb.Config{
		APIKey:    cfg.TMDB.APIKey,
		BaseURL:   cfg.TMDB.BaseURL,
		Language:  cfg.TMDB.Language,
		CacheTTL:  cfg.TMDB.CacheTTL,
		Transport: tmdbTransport(meters),
	})
	tmdbSource := tmdb.NewSource(tmdbClient)

	presets := preset.NewRegistry(preset.Builtins())
	overlay, err := preset.LoadOverlay(filepath.Join(cfg.ConfigDir, "presets.yml"))
	if err != nil {
		return nil, err
	}
	presets.Apply(overlay)
	if len(overlay) > 0 {
		logger.Info("user presets loaded", "count", len(overlay))
	}

	posters, err := poster.NewStore(poster.Config{
		BaseURL:      cfg.TMDB.ImageBaseURL,
		Dir:          filepath.Join(cfg.ConfigDir, "cache", "images"),
		CacheBytes:   cfg.Image.CacheBytes(),
		MaxFileBytes: cfg.Image.MaxFileBytes(),
	}, logger, posterCacheObserver(meters))
	if err != nil {
		return nil, err
	}

	decks := deck.NewBuilder(presets, deck.NewRegistry(tmdbSource), logger)
	sessions := session.New(decks, sessionLimits(cfg), logger)
	catalogSvc := catalog.New(presets, tmdbSource, sessions)

	var requester request.Requester
	if cfg.Seerr.URL != "" {
		requester = seerr.New(cfg.Seerr.URL, cfg.Seerr.APIKey)
		logger.Info("Seerr integration enabled", "url", cfg.Seerr.URL)
	}
	requests := request.New(sessions, requester, logger)

	hub := ws.NewHub(logger, sessions, ws.Limits{
		ConnsPerClient:  cfg.Limits.ConnsPerClient,
		ConnsPerSession: cfg.Limits.ConnsPerSession,
	})
	sessions.Subscribe(hub.HandleEvent)

	if meters != nil {
		sessions.Subscribe(func(_ string, ev session.Event) {
			meters.ObserveSessionEvent(string(ev.Type))
		})
		meters.SetSource(func() metrics.Snapshot {
			return snapshot(sessions.Stats(), hub.Connections())
		})
		for name, c := range tmdbClient.Caches() {
			meters.TrackCache(name, c.Hits, c.Misses)
		}
	}

	if cfg.Limits.On() {
		logger.Info("limits enabled", "sessions", cfg.Limits.Sessions, "clientIPHeader", cfg.Limits.ClientIPHeader)
	}

	server := transport.New(transport.Config{
		Port:              cfg.Server.Port,
		URLBase:           cfg.Server.URLBase,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		BodyLimit:         cfg.Limits.BodyBytes,
	}, transport.Deps{
		Logger:          logger,
		Sessions:        sessions,
		Catalog:         catalogSvc,
		Requests:        requests,
		Posters:         posters,
		Hub:             hub,
		Static:          web.DistFS(),
		RequestsEnabled: requests.Enabled(),
		Limiters:        limiters(cfg.Limits),
		Observer:        httpObserver(meters),
		PosterObserver:  posterObserver(meters),
	})

	return &App{
		cfg:      cfg,
		logger:   logger,
		sessions: sessions,
		posters:  posters,
		server:   server,
		meters:   meters,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	go a.sessions.RunJanitor(ctx)
	go a.posters.RunJanitor(ctx)

	if a.meters != nil {
		go func() {
			err := a.meters.Serve(ctx, metrics.ServerConfig{
				Port: a.cfg.Metrics.Port,
				Path: a.cfg.Metrics.Path,
			}, a.logger)
			if err != nil {
				a.logger.Error("metrics listener", "err", err)
			}
		}()
	}

	return a.server.Run(ctx)
}

func snapshot(stats session.Stats, connections int) metrics.Snapshot {
	sessions := make(map[string]int, len(stats.Sessions))
	for status, count := range stats.Sessions {
		sessions[string(status)] = count
	}
	return metrics.Snapshot{
		Sessions:     sessions,
		Participants: stats.Participants,
		Connections:  connections,
	}
}
