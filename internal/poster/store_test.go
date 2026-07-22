package poster

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T, cacheBytes int64) (*Store, string) {
	t.Helper()
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("poster-bytes"))
	}))
	t.Cleanup(cdn.Close)

	dir := t.TempDir()
	store, err := NewStore(Config{
		BaseURL:      cdn.URL,
		Dir:          dir,
		CacheBytes:   cacheBytes,
		MaxFileBytes: 1 << 20,
	}, slog.New(slog.DiscardHandler), nil)
	if err != nil {
		t.Fatal(err)
	}
	return store, dir
}

func TestFetchCachesAndOpenServes(t *testing.T) {
	store, dir := testStore(t, 1<<20)

	data, contentType, err := store.Fetch(context.Background(), "w185", "a.jpg")
	if err != nil || string(data) != "poster-bytes" || contentType != "image/jpeg" {
		t.Fatalf("Fetch() = %q, %q, %v", data, contentType, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "w185_a.jpg")); err != nil {
		t.Fatalf("the poster was not cached: %v", err)
	}

	cached, ok := store.Open("w185", "a.jpg")
	if !ok {
		t.Fatal("Open() must find the cached poster")
	}
	defer cached.Close()
	content, err := io.ReadAll(cached.File)
	if err != nil || string(content) != "poster-bytes" {
		t.Fatalf("cached content = %q, %v", content, err)
	}
}

func TestOpenMissingPoster(t *testing.T) {
	store, _ := testStore(t, 1<<20)
	if _, ok := store.Open("w185", "missing.jpg"); ok {
		t.Fatal("Open() invented a poster")
	}
}

func TestSweepEvictsOldestFirst(t *testing.T) {
	store, dir := testStore(t, 25)

	old := filepath.Join(dir, "w185_old.jpg")
	fresh := filepath.Join(dir, "w185_new.jpg")
	if err := os.WriteFile(old, make([]byte, 20), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fresh, make([]byte, 20), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}

	store.sweep()

	if _, err := os.Stat(old); err == nil {
		t.Fatal("the oldest poster must be evicted")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("the newest poster must survive")
	}
}

func TestSweepDropsInterruptedDownloads(t *testing.T) {
	store, dir := testStore(t, 1<<20)

	stale := filepath.Join(dir, tempPrefix+"stale")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stale, past, past); err != nil {
		t.Fatal(err)
	}
	inflight := filepath.Join(dir, tempPrefix+"inflight")
	if err := os.WriteFile(inflight, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store.sweep()

	if _, err := os.Stat(stale); err == nil {
		t.Fatal("debris from a killed process must be dropped")
	}
	if _, err := os.Stat(inflight); err != nil {
		t.Fatal("an in-flight temp file must be left alone")
	}
}

func TestFetchUpstreamFailure(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(cdn.Close)

	store, err := NewStore(Config{BaseURL: cdn.URL, Dir: t.TempDir(), MaxFileBytes: 1 << 20},
		slog.New(slog.DiscardHandler), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Fetch(context.Background(), "w185", "a.jpg"); err == nil {
		t.Fatal("a CDN failure must surface")
	}
}
