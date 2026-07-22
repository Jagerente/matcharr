package deck

import (
	"context"

	"github.com/Jagerente/matcharr/internal/domain"
)

type Query struct {
	MediaType domain.MediaType
	Endpoint  string
	Params    map[string]string
	Filters   domain.Filters
}

type Page struct {
	Movies     []domain.Movie
	Page       int
	TotalPages int
}

type Source interface {
	ID() string
	FetchPage(ctx context.Context, q Query, page int) (Page, error)
}

type Registry struct {
	sources map[string]Source
}

func NewRegistry(sources ...Source) *Registry {
	r := &Registry{sources: make(map[string]Source, len(sources))}
	for _, s := range sources {
		r.sources[s.ID()] = s
	}
	return r
}

func (r *Registry) Get(id string) (Source, bool) {
	s, ok := r.sources[id]
	return s, ok
}
