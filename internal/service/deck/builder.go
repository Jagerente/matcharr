package deck

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/Jagerente/matcharr/internal/domain"
)

type PresetCatalog interface {
	Get(id string) (domain.Preset, bool)
}

type Request struct {
	PresetID  string
	Filters   domain.Filters
	StartPage int
	Want      int
	Exclude   map[int64]struct{}
}

type Result struct {
	Movies    []domain.Movie
	NextPage  int
	Exhausted bool
}

type Builder struct {
	presets PresetCatalog
	sources *Registry
	logger  *slog.Logger
}

func NewBuilder(presets PresetCatalog, sources *Registry, logger *slog.Logger) *Builder {
	return &Builder{presets: presets, sources: sources, logger: logger}
}

func (b *Builder) HasPreset(id string) bool {
	_, ok := b.presets.Get(id)
	return ok
}

func (b *Builder) Build(ctx context.Context, req Request) (Result, error) {
	p, ok := b.presets.Get(req.PresetID)
	if !ok {
		return Result{}, fmt.Errorf("preset %q not found", req.PresetID)
	}
	src, ok := b.sources.Get(p.Source)
	if !ok {
		return Result{}, fmt.Errorf("source %q is not wired", p.Source)
	}

	query := Query{
		MediaType: p.MediaType,
		Endpoint:  p.Endpoint,
		Params:    p.Params,
		Filters:   req.Filters,
	}

	seen := make(map[int64]struct{}, len(req.Exclude))
	maps.Copy(seen, req.Exclude)

	var out []domain.Movie
	page := req.StartPage
	for len(out) < req.Want {
		fetched, err := src.FetchPage(ctx, query, page)
		if err != nil {
			if len(out) > 0 {
				// A partial deck beats a failed start.
				b.logger.Warn("deck built partially", "preset", req.PresetID, "got", len(out), "err", err)
				return Result{Movies: out, NextPage: page}, nil
			}
			return Result{}, err
		}
		if len(fetched.Movies) == 0 {
			return Result{Movies: out, NextPage: page, Exhausted: true}, nil
		}
		for _, movie := range fetched.Movies {
			if _, dup := seen[movie.ID]; dup {
				continue
			}
			seen[movie.ID] = struct{}{}
			out = append(out, movie)
			if len(out) == req.Want {
				break
			}
		}
		// A page may only be turned once everything on it is consumed: if
		// the limit filled mid-page, the next round picks up the rest.
		if pageConsumed(fetched.Movies, seen) {
			page++
			if fetched.TotalPages > 0 && page > fetched.TotalPages {
				return Result{Movies: out, NextPage: page, Exhausted: true}, nil
			}
		}
	}
	return Result{Movies: out, NextPage: page}, nil
}

func pageConsumed(movies []domain.Movie, seen map[int64]struct{}) bool {
	for _, movie := range movies {
		if _, taken := seen[movie.ID]; !taken {
			return false
		}
	}
	return true
}
