package poster

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	downloadTimeout = 30 * time.Second
	tempPrefix      = "dl-"
)

type Config struct {
	BaseURL string
	Dir     string
	// CacheBytes is the budget for the whole cache directory, MaxFileBytes
	// the ceiling on one download: the CDN response is read into memory
	// before it is served.
	CacheBytes   int64
	MaxFileBytes int64
}

// Observer sees what each sweep cost. Nil is a valid value: an instance
// without metrics wired pays nothing.
type Observer interface {
	ObservePosterCache(size int64, evicted int)
}

// Store caches posters on disk under names the transport layer has already
// validated; it never derives paths from raw request input itself.
type Store struct {
	baseURL  string
	dir      string
	maxBytes int64
	maxFile  int64
	client   *http.Client
	logger   *slog.Logger
	observer Observer
}

func NewStore(cfg Config, logger *slog.Logger, observer Observer) (*Store, error) {
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("poster cache: %w", err)
	}
	return &Store{
		baseURL:  cfg.BaseURL,
		dir:      cfg.Dir,
		maxBytes: cfg.CacheBytes,
		maxFile:  cfg.MaxFileBytes,
		client:   &http.Client{Timeout: downloadTimeout},
		logger:   logger,
		observer: observer,
	}, nil
}

type Cached struct {
	File *os.File
	Info fs.FileInfo
}

func (c *Cached) Close() { _ = c.File.Close() }

// Open returns the cached poster with the file already open: a janitor
// sweep landing between a lookup and a later read would otherwise turn a
// cached poster into a 404.
func (st *Store) Open(size, file string) (*Cached, bool) {
	f, err := os.Open(st.path(size, file))
	if err != nil {
		return nil, false
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, false
	}
	return &Cached{File: f, Info: info}, true
}

func (st *Store) Fetch(ctx context.Context, size, file string) (data []byte, contentType string, err error) {
	upstream := st.baseURL + "/" + size + "/" + file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream, nil)
	if err != nil {
		return nil, "", fmt.Errorf("poster: %w", err)
	}

	resp, err := st.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("poster: cdn unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("poster: cdn responded with %s", resp.Status)
	}

	data, err = io.ReadAll(io.LimitReader(resp.Body, st.maxFile))
	if err != nil {
		return nil, "", fmt.Errorf("poster: cdn aborted the response: %w", err)
	}

	st.cache(size, file, data)
	return data, resp.Header.Get("Content-Type"), nil
}

// cache is best-effort: temp file plus rename so concurrent requests never
// read a half-written file, and a write failure still serves the image.
func (st *Store) cache(size, file string, data []byte) {
	tmp, err := os.CreateTemp(st.dir, tempPrefix+"*")
	if err != nil {
		return
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return
	}
	_ = tmp.Close()
	_ = os.Rename(tmp.Name(), st.path(size, file))
}

func (st *Store) path(size, file string) string {
	return filepath.Join(st.dir, size+"_"+file)
}
