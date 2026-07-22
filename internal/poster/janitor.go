package poster

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const sweepInterval = time.Hour

func (st *Store) RunJanitor(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	st.sweep()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.sweep()
		}
	}
}

// sweep holds the cache within its byte budget and evicts the posters
// fetched longest ago first: mtime is the only age signal that survives
// container volumes, atime is disabled or coarse on most of them.
func (st *Store) sweep() {
	files, total := st.scan()
	if st.maxBytes <= 0 || total <= st.maxBytes {
		st.observeCache(total, 0)
		return
	}
	slices.SortFunc(files, func(a, b cachedFile) int { return a.modTime.Compare(b.modTime) })

	var removed int
	var freed int64
	for _, f := range files {
		if total-freed <= st.maxBytes {
			break
		}
		if err := os.Remove(filepath.Join(st.dir, f.name)); err != nil {
			st.logger.Warn("poster cache: evicting", "file", f.name, "err", err)
			continue
		}
		removed++
		freed += f.size
	}
	st.observeCache(total-freed, removed)
	st.logger.Info("poster cache trimmed", "removed", removed, "freed", freed, "budget", st.maxBytes)
}

func (st *Store) observeCache(size int64, evicted int) {
	if st.observer != nil {
		st.observer.ObservePosterCache(size, evicted)
	}
}

type cachedFile struct {
	name    string
	size    int64
	modTime time.Time
}

func (st *Store) scan() ([]cachedFile, int64) {
	entries, err := os.ReadDir(st.dir)
	if err != nil {
		st.logger.Error("poster cache: reading the cache directory", "err", err)
		return nil, 0
	}

	files := make([]cachedFile, 0, len(entries))
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			// The entry is already gone: a concurrent download renamed its
			// temp file over that name between the listing and the stat.
			continue
		}
		if strings.HasPrefix(e.Name(), tempPrefix) {
			st.dropInterrupted(e.Name(), info.ModTime())
			continue
		}
		files = append(files, cachedFile{name: e.Name(), size: info.Size(), modTime: info.ModTime()})
		total += info.Size()
	}
	return files, total
}

// dropInterrupted removes debris from a killed process: a temp file is
// renamed into place right after the body is read, so one older than the
// download timeout cannot belong to a live write. Younger ones are left
// alone: they may belong to an in-flight one.
func (st *Store) dropInterrupted(name string, modTime time.Time) {
	if time.Since(modTime) < downloadTimeout {
		return
	}
	if err := os.Remove(filepath.Join(st.dir, name)); err != nil {
		st.logger.Warn("poster cache: removing an interrupted download", "file", name, "err", err)
	}
}
