package config

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Jagerente/gocfg"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MATCHARR__TMDB__APIKEY", "key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.ConfigDir != "./config" {
		t.Errorf("ConfigDir = %q, want ./config", cfg.ConfigDir)
	}
	if cfg.Server.Port != 8585 {
		t.Errorf("Server.Port = %d, want 8585", cfg.Server.Port)
	}
	if cfg.Server.URLBase != "" {
		t.Errorf("Server.URLBase = %q, want empty", cfg.Server.URLBase)
	}
	if cfg.TMDB.CacheTTL != 30*time.Minute {
		t.Errorf("TMDB.CacheTTL = %s, want 30m", cfg.TMDB.CacheTTL)
	}
	if cfg.Image.CacheBytes() != 512<<20 {
		t.Errorf("Image.CacheBytes() = %d, want %d", cfg.Image.CacheBytes(), 512<<20)
	}
	if cfg.Session.TTL != 24*time.Hour {
		t.Errorf("Session.TTL = %s, want 24h", cfg.Session.TTL)
	}
	if cfg.Log.Level != slog.LevelInfo {
		t.Errorf("Log.Level = %s, want info", cfg.Log.Level)
	}
	if cfg.Limits.On() {
		t.Errorf("Limits = %+v, a default install must stay unthrottled", cfg.Limits)
	}
}

func TestLoadLimitKeys(t *testing.T) {
	env := map[string]string{
		"MATCHARR__TMDB__APIKEY":            "key",
		"MATCHARR__LIMITS__SESSIONS":        "200",
		"MATCHARR__LIMITS__PARTICIPANTS":    "8",
		"MATCHARR__LIMITS__DECK":            "120",
		"MATCHARR__LIMITS__CONNSPERCLIENT":  "2",
		"MATCHARR__LIMITS__CONNSPERSESSION": "20",
		"MATCHARR__LIMITS__BODYBYTES":       "4096",
		"MATCHARR__LIMITS__CLIENTIPHEADER":  "CF-Connecting-IP",
		"MATCHARR__LIMITS__CREATEBURST":     "5",
		"MATCHARR__LIMITS__CREATEEVERY":     "2m",
		"MATCHARR__LIMITS__JOINBURST":       "10",
		"MATCHARR__LIMITS__JOINEVERY":       "10s",
		"MATCHARR__LIMITS__ACTIONBURST":     "60",
		"MATCHARR__LIMITS__ACTIONEVERY":     "200ms",
		"MATCHARR__LIMITS__CATALOGBURST":    "30",
		"MATCHARR__LIMITS__CATALOGEVERY":    "2s",
		"MATCHARR__LIMITS__IMAGEBURST":      "150",
		"MATCHARR__LIMITS__IMAGEEVERY":      "250ms",
	}
	for key, value := range env {
		t.Setenv(key, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	want := Limits{
		Sessions:        200,
		Participants:    8,
		Deck:            120,
		ConnsPerClient:  2,
		ConnsPerSession: 20,
		BodyBytes:       4096,
		ClientIPHeader:  "CF-Connecting-IP",
		Create:          Rate{Burst: 5, Every: 2 * time.Minute},
		Join:            Rate{Burst: 10, Every: 10 * time.Second},
		Action:          Rate{Burst: 60, Every: 200 * time.Millisecond},
		Catalog:         Rate{Burst: 30, Every: 2 * time.Second},
		Image:           Rate{Burst: 150, Every: 250 * time.Millisecond},
	}
	if cfg.Limits != want {
		t.Errorf("Limits = %+v, want %+v", cfg.Limits, want)
	}
}

func TestLoadRejectsInstantReaping(t *testing.T) {
	for _, key := range []string{"TTL", "EMPTYGRACE", "FINISHEDTTL"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("MATCHARR__TMDB__APIKEY", "key")
			t.Setenv("MATCHARR__SESSION__"+key, "0s")

			if _, err := Load(); err == nil {
				t.Fatalf("MATCHARR__SESSION__%s: a session lifetime of zero must not start", key)
			}
		})
	}
}

func TestLoadRejectsUnrefillableBucket(t *testing.T) {
	t.Setenv("MATCHARR__TMDB__APIKEY", "key")
	t.Setenv("MATCHARR__LIMITS__ACTIONBURST", "10")
	t.Setenv("MATCHARR__LIMITS__ACTIONEVERY", "0s")

	if _, err := Load(); err == nil {
		t.Fatal("a bucket that never refills must not start")
	}
}

// The envPrefix tags must reproduce the documented MATCHARR__SECTION__KEY scheme
// exactly: the keys are part of the deployment contract, not an implementation
// detail.
func TestLoadPrefixedKeys(t *testing.T) {
	env := map[string]string{
		"MATCHARR__CONFIG__DIR":          "/config",
		"MATCHARR__SERVER__PORT":         "9000",
		"MATCHARR__TMDB__APIKEY":         "key",
		"MATCHARR__TMDB__BASEURL":        "http://tmdb.local",
		"MATCHARR__TMDB__IMAGEBASEURL":   "http://images.local/t/p",
		"MATCHARR__TMDB__LANGUAGE":       "ru-RU",
		"MATCHARR__TMDB__CACHETTL":       "5m",
		"MATCHARR__IMAGE__CACHESIZEMB":   "64",
		"MATCHARR__SEERR__URL":           "http://seerr:5055",
		"MATCHARR__SEERR__APIKEY":        "seerr-key",
		"MATCHARR__RADARR__URL":          "http://radarr:7878",
		"MATCHARR__RADARR__APIKEY":       "radarr-key",
		"MATCHARR__RADARR__ROOTFOLDER":   "/movies",
		"MATCHARR__SESSION__TTL":         "1h",
		"MATCHARR__SESSION__EMPTYGRACE":  "5m",
		"MATCHARR__SESSION__FINISHEDTTL": "20m",
		"MATCHARR__LOG__LEVEL":           "debug",
	}
	for key, value := range env {
		t.Setenv(key, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	got := map[string]any{
		"MATCHARR__CONFIG__DIR":          cfg.ConfigDir,
		"MATCHARR__SERVER__PORT":         cfg.Server.Port,
		"MATCHARR__TMDB__APIKEY":         cfg.TMDB.APIKey,
		"MATCHARR__TMDB__BASEURL":        cfg.TMDB.BaseURL,
		"MATCHARR__TMDB__IMAGEBASEURL":   cfg.TMDB.ImageBaseURL,
		"MATCHARR__TMDB__LANGUAGE":       cfg.TMDB.Language,
		"MATCHARR__TMDB__CACHETTL":       cfg.TMDB.CacheTTL,
		"MATCHARR__IMAGE__CACHESIZEMB":   cfg.Image.CacheSizeMB,
		"MATCHARR__SEERR__URL":           cfg.Seerr.URL,
		"MATCHARR__SEERR__APIKEY":        cfg.Seerr.APIKey,
		"MATCHARR__RADARR__URL":          cfg.Radarr.URL,
		"MATCHARR__RADARR__APIKEY":       cfg.Radarr.APIKey,
		"MATCHARR__RADARR__ROOTFOLDER":   cfg.Radarr.RootFolder,
		"MATCHARR__SESSION__TTL":         cfg.Session.TTL,
		"MATCHARR__SESSION__EMPTYGRACE":  cfg.Session.EmptyGrace,
		"MATCHARR__SESSION__FINISHEDTTL": cfg.Session.FinishedTTL,
		"MATCHARR__LOG__LEVEL":           cfg.Log.Level,
	}
	want := map[string]any{
		"MATCHARR__CONFIG__DIR":          "/config",
		"MATCHARR__SERVER__PORT":         9000,
		"MATCHARR__TMDB__APIKEY":         "key",
		"MATCHARR__TMDB__BASEURL":        "http://tmdb.local",
		"MATCHARR__TMDB__IMAGEBASEURL":   "http://images.local/t/p",
		"MATCHARR__TMDB__LANGUAGE":       "ru-RU",
		"MATCHARR__TMDB__CACHETTL":       5 * time.Minute,
		"MATCHARR__IMAGE__CACHESIZEMB":   int64(64),
		"MATCHARR__SEERR__URL":           "http://seerr:5055",
		"MATCHARR__SEERR__APIKEY":        "seerr-key",
		"MATCHARR__RADARR__URL":          "http://radarr:7878",
		"MATCHARR__RADARR__APIKEY":       "radarr-key",
		"MATCHARR__RADARR__ROOTFOLDER":   "/movies",
		"MATCHARR__SESSION__TTL":         time.Hour,
		"MATCHARR__SESSION__EMPTYGRACE":  5 * time.Minute,
		"MATCHARR__SESSION__FINISHEDTTL": 20 * time.Minute,
		"MATCHARR__LOG__LEVEL":           slog.LevelDebug,
	}

	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Errorf("%s: got %v, want %v", key, got[key], wantValue)
		}
	}
}

func TestLoadMissingAPIKey(t *testing.T) {
	t.Setenv("MATCHARR__TMDB__APIKEY", "")

	_, err := Load()
	if !errors.Is(err, gocfg.ErrRequired) {
		t.Fatalf("Load() = %v, want gocfg.ErrRequired", err)
	}

	var fieldErr *gocfg.FieldError
	if !errors.As(err, &fieldErr) {
		t.Fatalf("Load() = %v, want a *gocfg.FieldError", err)
	}
	if fieldErr.Key != "MATCHARR__TMDB__APIKEY" {
		t.Errorf("FieldError.Key = %q, want MATCHARR__TMDB__APIKEY", fieldErr.Key)
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	t.Setenv("MATCHARR__TMDB__APIKEY", "key")
	t.Setenv("MATCHARR__LOG__LEVEL", "chatty")

	if _, err := Load(); !errors.Is(err, gocfg.ErrParse) {
		t.Fatalf("Load() = %v, want gocfg.ErrParse", err)
	}
}

func TestLoadURLBase(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "bare", raw: "matcharr", want: "/matcharr"},
		{name: "slashed", raw: "/matcharr/", want: "/matcharr"},
		{name: "nested", raw: "/apps/matcharr", want: "/apps/matcharr"},
		{name: "slashes only", raw: "///", want: ""},
		{name: "space", raw: "match arr", wantErr: true},
		{name: "query", raw: "matcharr?x=1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MATCHARR__TMDB__APIKEY", "key")
			t.Setenv("MATCHARR__SERVER__URLBASE", tt.raw)

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() = nil, want an error for %q", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() = %v", err)
			}
			if cfg.Server.URLBase != tt.want {
				t.Errorf("Server.URLBase = %q, want %q", cfg.Server.URLBase, tt.want)
			}
		})
	}
}
