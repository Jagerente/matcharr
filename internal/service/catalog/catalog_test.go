package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

type fakePresets []domain.Preset

func (f fakePresets) List() []domain.Preset { return f }

type fakeMovies struct {
	genres []domain.Genre
	movie  domain.Movie
	err    error
}

func (f *fakeMovies) Genres(context.Context) ([]domain.Genre, error) { return f.genres, f.err }

func (f *fakeMovies) Details(context.Context, int64) (domain.Movie, error) { return f.movie, f.err }

type fakeAuthorizer struct{ err error }

func (f *fakeAuthorizer) AuthorizeMovie(_, _ string, _ int64) error { return f.err }

func TestGenresWrapsUpstreamFailure(t *testing.T) {
	c := New(fakePresets{}, &fakeMovies{err: errors.New("down")}, &fakeAuthorizer{})

	_, err := c.Genres(context.Background())
	appErr := apperr.From(err)
	if appErr == nil || appErr.Reason != apperr.ReasonSourceUnavailable {
		t.Fatalf("err = %v, want source_unavailable", err)
	}
}

func TestMovieDetailsRequiresAuthorization(t *testing.T) {
	authErr := apperr.New(apperr.KindInvalid, apperr.ReasonUnknownMovie, "not dealt")
	c := New(fakePresets{}, &fakeMovies{movie: domain.Movie{ID: 5}}, &fakeAuthorizer{err: authErr})

	if _, err := c.MovieDetails(context.Background(), "AAAA", "t", 5); !errors.Is(err, authErr) {
		t.Fatalf("err = %v, want the authorization error passed through", err)
	}
}

func TestMovieDetails(t *testing.T) {
	c := New(fakePresets{}, &fakeMovies{movie: domain.Movie{ID: 5, Title: "Heat"}}, &fakeAuthorizer{})

	movie, err := c.MovieDetails(context.Background(), "AAAA", "t", 5)
	if err != nil || movie.Title != "Heat" {
		t.Fatalf("MovieDetails() = %v, %v", movie, err)
	}
}
