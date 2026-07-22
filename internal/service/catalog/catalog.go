package catalog

import (
	"context"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

type PresetSource interface {
	List() []domain.Preset
}

type MovieSource interface {
	Genres(ctx context.Context) ([]domain.Genre, error)
	Details(ctx context.Context, id int64) (domain.Movie, error)
}

type MovieAuthorizer interface {
	AuthorizeMovie(code, token string, movieID int64) error
}

type Catalog struct {
	presets  PresetSource
	movies   MovieSource
	sessions MovieAuthorizer
}

func New(presets PresetSource, movies MovieSource, sessions MovieAuthorizer) *Catalog {
	return &Catalog{presets: presets, movies: movies, sessions: sessions}
}

func (c *Catalog) Presets() []domain.Preset { return c.presets.List() }

func (c *Catalog) Genres(ctx context.Context) ([]domain.Genre, error) {
	genres, err := c.movies.Genres(ctx)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.KindUpstream, apperr.ReasonSourceUnavailable, "the source is unavailable")
	}
	return genres, nil
}

// MovieDetails serves details only for movies the session actually dealt,
// so the endpoint cannot be used to browse the catalog anonymously.
func (c *Catalog) MovieDetails(ctx context.Context, code, token string, movieID int64) (domain.Movie, error) {
	if err := c.sessions.AuthorizeMovie(code, token, movieID); err != nil {
		return domain.Movie{}, err
	}
	movie, err := c.movies.Details(ctx, movieID)
	if err != nil {
		return domain.Movie{}, apperr.Wrap(err, apperr.KindUpstream, apperr.ReasonSourceUnavailable, "the source is unavailable")
	}
	return movie, nil
}
