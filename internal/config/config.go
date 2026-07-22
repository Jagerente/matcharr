package config

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/Jagerente/gocfg"
	"github.com/Jagerente/gocfg/pkg/parsers"
	"github.com/Jagerente/gocfg/pkg/values"
)

type Config struct {
	ConfigDir string `env:"MATCHARR__CONFIG__DIR" default:"./config"`

	Server  Server  `title:"Server" envPrefix:"MATCHARR__SERVER__"`
	TMDB    TMDB    `title:"TMDB" envPrefix:"MATCHARR__TMDB__"`
	Image   Image   `title:"Image" envPrefix:"MATCHARR__IMAGE__"`
	Seerr   Seerr   `title:"Seerr" envPrefix:"MATCHARR__SEERR__"`
	Session Session `title:"Session" envPrefix:"MATCHARR__SESSION__"`
	Limits  Limits  `title:"Limits" envPrefix:"MATCHARR__LIMITS__"`
	Metrics Metrics `title:"Metrics" envPrefix:"MATCHARR__METRICS__"`
	Log     Log     `title:"Log" envPrefix:"MATCHARR__LOG__"`
}

type Server struct {
	// Port defaults away from 8484, which Docker Desktop holds on the
	// target machine.
	Port    int    `env:"PORT" default:"8585"`
	URLBase string `env:"URLBASE,omitempty"`
	// ReadTimeout, ReadHeaderTimeout and IdleTimeout bound a request, its
	// headers alone and an idle keep-alive connection. Zero disables one,
	// which is what a trusted network can afford; the event socket is
	// exempt either way.
	ReadTimeout       time.Duration `env:"READTIMEOUT" default:"30s"`
	ReadHeaderTimeout time.Duration `env:"READHEADERTIMEOUT" default:"10s"`
	IdleTimeout       time.Duration `env:"IDLETIMEOUT" default:"2m"`
}

type TMDB struct {
	APIKey       string        `env:"APIKEY"`
	BaseURL      string        `env:"BASEURL" default:"https://api.themoviedb.org"`
	ImageBaseURL string        `env:"IMAGEBASEURL" default:"https://image.tmdb.org/t/p"`
	Language     string        `env:"LANGUAGE" default:"en-US"`
	CacheTTL     time.Duration `env:"CACHETTL" default:"30m"`
}

type Image struct {
	CacheSizeMB int64 `env:"CACHESIZEMB" default:"512"`
	MaxFileMB   int64 `env:"MAXFILEMB" default:"4"`
}

func (i Image) CacheBytes() int64   { return i.CacheSizeMB << 20 }
func (i Image) MaxFileBytes() int64 { return i.MaxFileMB << 20 }

type Seerr struct {
	URL    string `env:"URL,omitempty"`
	APIKey string `env:"APIKEY,omitempty"`
}

type Session struct {
	TTL         time.Duration `env:"TTL" default:"24h"`
	EmptyGrace  time.Duration `env:"EMPTYGRACE" default:"15m"`
	FinishedTTL time.Duration `env:"FINISHEDTTL" default:"1h"`
}

func (s Session) validate() error {
	spans := []struct {
		key   string
		value time.Duration
	}{
		{"TTL", s.TTL},
		{"EMPTYGRACE", s.EmptyGrace},
		{"FINISHEDTTL", s.FinishedTTL},
	}
	for _, span := range spans {
		if span.value <= 0 {
			return fmt.Errorf("MATCHARR__SESSION__%s: %s would reap sessions the moment they open", span.key, span.value)
		}
	}
	return nil
}

// Limits are the guards for an instance strangers can reach. Everything here
// is off by default, which is the LAN and Tailscale case.
type Limits struct {
	// Sessions is the ceiling on live sessions, 0 for no ceiling.
	Sessions int `env:"SESSIONS" default:"0"`
	// Participants is the ceiling on people in one session, Deck on movies
	// dealt per round: together they bound what a single session can cost.
	Participants int `env:"PARTICIPANTS" default:"16"`
	Deck         int `env:"DECK" default:"200"`
	// ConnsPerClient and ConnsPerSession cap the event sockets one
	// participant and one session may hold open. A phone that reconnects
	// before the old socket is reaped needs more than one.
	ConnsPerClient  int `env:"CONNSPERCLIENT" default:"3"`
	ConnsPerSession int `env:"CONNSPERSESSION" default:"48"`
	// BodyBytes caps a request body. Every endpoint here takes a handful of
	// short fields, so anything larger is someone probing how much memory a
	// stranger can make us allocate.
	BodyBytes int64 `env:"BODYBYTES" default:"8192"`
	// ClientIPHeader is where a trusted reverse proxy puts the real client
	// address (CF-Connecting-IP behind Cloudflare). Setting it while the
	// proxy does not overwrite that header lets clients forge their identity
	// and every rate limit becomes optional.
	ClientIPHeader string `env:"CLIENTIPHEADER,omitempty"`

	Create  Rate `envPrefix:"CREATE"`
	Join    Rate `envPrefix:"JOIN"`
	Action  Rate `envPrefix:"ACTION"`
	Catalog Rate `envPrefix:"CATALOG"`
	Image   Rate `envPrefix:"IMAGE"`
}

// Rate is a token bucket per client address: Burst requests back to back, then
// one more every Every. A zero burst leaves those routes unlimited.
type Rate struct {
	Burst int           `env:"BURST" default:"0"`
	Every time.Duration `env:"EVERY" default:"1s"`
}

func (r Rate) On() bool { return r.Burst > 0 }

func (l Limits) On() bool {
	if l.Sessions > 0 {
		return true
	}
	for _, r := range l.rates() {
		if r.rate.On() {
			return true
		}
	}
	return false
}

type namedRate struct {
	key  string
	rate Rate
}

func (l Limits) rates() []namedRate {
	return []namedRate{
		{"CREATE", l.Create},
		{"JOIN", l.Join},
		{"ACTION", l.Action},
		{"CATALOG", l.Catalog},
		{"IMAGE", l.Image},
	}
}

func (l Limits) validate() error {
	bounds := []struct {
		key      string
		value    int64
		smallest int64
	}{
		{"SESSIONS", int64(l.Sessions), 0},
		{"PARTICIPANTS", int64(l.Participants), 2},
		{"DECK", int64(l.Deck), 1},
		{"CONNSPERCLIENT", int64(l.ConnsPerClient), 1},
		{"CONNSPERSESSION", int64(l.ConnsPerSession), 1},
		{"BODYBYTES", l.BodyBytes, 1},
	}
	for _, b := range bounds {
		if b.value < b.smallest {
			return fmt.Errorf("MATCHARR__LIMITS__%s: %d is below the minimum of %d", b.key, b.value, b.smallest)
		}
	}

	for _, r := range l.rates() {
		if r.rate.Burst < 0 {
			return fmt.Errorf("MATCHARR__LIMITS__%sBURST: %d is negative", r.key, r.rate.Burst)
		}
		if r.rate.On() && r.rate.Every <= 0 {
			return fmt.Errorf("MATCHARR__LIMITS__%sEVERY: %s leaves the bucket unable to refill", r.key, r.rate.Every)
		}
	}
	return nil
}

func (i Image) validate() error {
	if i.CacheSizeMB < 0 {
		return fmt.Errorf("MATCHARR__IMAGE__CACHESIZEMB: %d is negative", i.CacheSizeMB)
	}
	if i.MaxFileMB < 1 {
		return fmt.Errorf("MATCHARR__IMAGE__MAXFILEMB: %d leaves no room for a poster", i.MaxFileMB)
	}
	return nil
}

// Metrics are served on a port of their own, never the one the app is
// published on: the exposition is unauthenticated and describes the instance.
// Port 0 keeps them off.
type Metrics struct {
	Port int    `env:"PORT" default:"0"`
	Path string `env:"PATH" default:"/metrics"`
}

func (m Metrics) On() bool { return m.Port > 0 }

func (m Metrics) validate() error {
	if m.Port < 0 || m.Port > 65535 {
		return fmt.Errorf("MATCHARR__METRICS__PORT: %d is not a port", m.Port)
	}
	if m.On() && !strings.HasPrefix(m.Path, "/") {
		return fmt.Errorf("MATCHARR__METRICS__PATH: %q must start with a slash", m.Path)
	}
	return nil
}

type Log struct {
	Level slog.Level `env:"LEVEL" default:"info"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	// The TextUnmarshaler provider has to outrank the built-in ones: slog.Level
	// is an int kind, so the kind parser would reject "info" before
	// UnmarshalText ever gets a chance to decode it.
	manager := gocfg.NewEmpty().
		AddParserProviders(
			parsers.NewTextUnmarshalerParserProvider(),
			parsers.NewDefaultParserProvider(),
		).
		AddValueProviders(values.NewEnvProvider()).
		UseDefaults().
		CollectAllErrors()

	if err := manager.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if err := cfg.Session.validate(); err != nil {
		return nil, err
	}
	if err := cfg.Limits.validate(); err != nil {
		return nil, err
	}
	if err := cfg.Image.validate(); err != nil {
		return nil, err
	}
	if err := cfg.Metrics.validate(); err != nil {
		return nil, err
	}

	base := strings.Trim(cfg.Server.URLBase, "/")
	if base == "" {
		cfg.Server.URLBase = ""
		return cfg, nil
	}

	if !urlBaseRe.MatchString(base) {
		// A malformed base path would corrupt every route registered under it.
		return nil, fmt.Errorf("invalid MATCHARR__SERVER__URLBASE: %q", cfg.Server.URLBase)
	}
	cfg.Server.URLBase = "/" + base

	return cfg, nil
}

var urlBaseRe = regexp.MustCompile(`^[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*$`)
