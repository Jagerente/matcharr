package app

import (
	"net/http"

	"github.com/Jagerente/matcharr/internal/config"
	"github.com/Jagerente/matcharr/internal/metrics"
	"github.com/Jagerente/matcharr/internal/poster"
	"github.com/Jagerente/matcharr/internal/ratelimit"
	"github.com/Jagerente/matcharr/internal/service/session"
	transport "github.com/Jagerente/matcharr/internal/transport/http"
	"github.com/Jagerente/matcharr/internal/transport/http/api"
	"github.com/Jagerente/matcharr/internal/transport/http/middleware"
)

func sessionLimits(cfg *config.Config) session.Limits {
	return session.Limits{
		TTL:             cfg.Session.TTL,
		EmptyGrace:      cfg.Session.EmptyGrace,
		FinishedTTL:     cfg.Session.FinishedTTL,
		MaxSessions:     cfg.Limits.Sessions,
		MaxParticipants: cfg.Limits.Participants,
		MaxDeck:         cfg.Limits.Deck,
	}
}

func limiters(cfg config.Limits) transport.Limiters {
	return transport.Limiters{
		Create:   limiter(cfg.Create),
		Join:     limiter(cfg.Join),
		Action:   limiter(cfg.Action),
		Catalog:  limiter(cfg.Catalog),
		Image:    limiter(cfg.Image),
		ClientIP: middleware.ClientIP{Header: cfg.ClientIPHeader},
	}
}

func limiter(rate config.Rate) middleware.RateLimiter {
	if !rate.On() {
		return nil
	}
	return ratelimit.New(rate.Burst, rate.Every)
}

// httpObserver and the wrappers below keep a missing metrics dependency
// as an honest nil: a typed nil pointer would satisfy the interface and
// defeat the nil checks downstream.
func httpObserver(m *metrics.Metrics) middleware.Observer {
	if m == nil {
		return nil
	}
	return m
}

func posterObserver(m *metrics.Metrics) api.PosterObserver {
	if m == nil {
		return nil
	}
	return m
}

func posterCacheObserver(m *metrics.Metrics) poster.Observer {
	if m == nil {
		return nil
	}
	return m
}

func tmdbTransport(m *metrics.Metrics) http.RoundTripper {
	if m == nil {
		return nil
	}
	return m.Transport(nil)
}
