package api

import (
	"context"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/poster"
)

// posterFileRe allows a single path-free file name only: it guards
// against path traversal.
var posterFileRe = regexp.MustCompile(`^[A-Za-z0-9]+\.(?:jpg|jpeg|png|webp)$`)

var posterSizes = map[string]struct{}{
	"w185": {}, "w500": {},
}

type PosterStore interface {
	Open(size, file string) (*poster.Cached, bool)
	Fetch(ctx context.Context, size, file string) (data []byte, contentType string, err error)
}

// PosterObserver sees how each poster was served. Nil is a valid value: an
// instance without metrics wired pays nothing.
type PosterObserver interface {
	ObservePoster(result string, bytes int)
}

const (
	posterHit   = "hit"
	posterMiss  = "miss"
	posterError = "error"
)

type PosterHandler struct {
	store    PosterStore
	observer PosterObserver
}

func NewPosterHandler(store PosterStore, observer PosterObserver) *PosterHandler {
	return &PosterHandler{store: store, observer: observer}
}

func (h *PosterHandler) Serve(c echo.Context) error {
	size, file := c.Param("size"), c.Param("file")
	if _, ok := posterSizes[size]; !ok {
		return apperr.New(apperr.KindInvalid, apperr.ReasonBadRequest, "unsupported size")
	}
	if !posterFileRe.MatchString(file) {
		return apperr.New(apperr.KindInvalid, apperr.ReasonBadRequest, "invalid file name")
	}

	if cached, ok := h.store.Open(size, file); ok {
		defer cached.Close()
		setPosterHeaders(c)
		http.ServeContent(c.Response(), c.Request(), file, cached.Info.ModTime(), cached.File)
		h.observe(posterHit, int(cached.Info.Size()))
		return nil
	}

	data, contentType, err := h.store.Fetch(c.Request().Context(), size, file)
	if err != nil {
		h.observe(posterError, 0)
		return apperr.Wrap(err, apperr.KindUpstream, apperr.ReasonSourceUnavailable, "fetching the poster")
	}

	setPosterHeaders(c)
	if contentType != "" {
		c.Response().Header().Set(echo.HeaderContentType, contentType)
	}
	if _, err := c.Response().Write(data); err != nil {
		return nil
	}
	h.observe(posterMiss, len(data))
	return nil
}

func (h *PosterHandler) observe(result string, bytes int) {
	if h.observer != nil {
		h.observer.ObservePoster(result, bytes)
	}
}

func setPosterHeaders(c echo.Context) {
	c.Response().Header().Set("Cache-Control", "public, max-age=604800, immutable")
}
