package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/deck"
)

const SourceID = "tmdb"

const (
	EndpointDiscover   = "discover"
	EndpointTrending   = "trending"
	EndpointPopular    = "popular"
	EndpointTopRated   = "top_rated"
	EndpointNowPlaying = "now_playing"
)

type Source struct {
	client *Client
	now    func() time.Time
}

func NewSource(client *Client) *Source {
	return &Source{client: client, now: time.Now}
}

func (s *Source) ID() string { return SourceID }

func (s *Source) FetchPage(ctx context.Context, q deck.Query, page int) (deck.Page, error) {
	if q.MediaType != "" && q.MediaType != domain.MediaTypeMovie {
		return deck.Page{}, fmt.Errorf("tmdb: unsupported media type %q", q.MediaType)
	}

	path, params, postFilter, err := s.resolve(q)
	if err != nil {
		return deck.Page{}, err
	}

	resp, err := s.client.moviesPage(ctx, path, params, page)
	if err != nil {
		return deck.Page{}, err
	}

	// Genres are display-only and must not fail the page.
	genres, _ := s.client.movieGenres(ctx)

	movies := make([]domain.Movie, 0, len(resp.Results))
	for _, dto := range resp.Results {
		// List endpoints cannot filter server-side, so their results are
		// filtered locally; the deck builder pages until it has enough.
		if postFilter && !dtoMatches(dto, q.Filters) {
			continue
		}
		movies = append(movies, toDomain(dto, genres))
	}

	return deck.Page{
		Movies:     movies,
		Page:       resp.Page,
		TotalPages: resp.TotalPages,
	}, nil
}

func (s *Source) Details(ctx context.Context, id int64) (domain.Movie, error) {
	dto, err := s.client.movieDetails(ctx, id)
	if err != nil {
		return domain.Movie{}, err
	}
	return detailsToDomain(dto), nil
}

func (s *Source) Genres(ctx context.Context) ([]domain.Genre, error) {
	byID, err := s.client.movieGenres(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Genre, 0, len(byID))
	for id, name := range byID {
		out = append(out, domain.Genre{ID: id, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Source) resolve(q deck.Query) (path string, params url.Values, postFilter bool, err error) {
	params = url.Values{}
	for k, v := range q.Params {
		params.Set(k, resolveRelativeDate(v, s.now()))
	}

	switch q.Endpoint {
	case EndpointDiscover, "":
		// Filters override preset params on collision: user intent wins.
		applyFilterParams(params, q.Filters)
		return "/discover/movie", params, false, nil
	case EndpointTrending:
		window := params.Get("window")
		if window == "" {
			window = "week"
		}
		params.Del("window")
		return "/trending/movie/" + window, params, !q.Filters.IsZero(), nil
	case EndpointPopular:
		return "/movie/popular", params, !q.Filters.IsZero(), nil
	case EndpointTopRated:
		return "/movie/top_rated", params, !q.Filters.IsZero(), nil
	case EndpointNowPlaying:
		return "/movie/now_playing", params, !q.Filters.IsZero(), nil
	default:
		return "", nil, false, fmt.Errorf("tmdb: unknown endpoint %q", q.Endpoint)
	}
}

func applyFilterParams(params url.Values, f domain.Filters) {
	if len(f.GenreIDs) > 0 {
		ids := make([]string, len(f.GenreIDs))
		for i, id := range f.GenreIDs {
			ids[i] = strconv.FormatInt(id, 10)
		}
		// A pipe means OR in with_genres.
		params.Set("with_genres", strings.Join(ids, "|"))
	}
	if f.YearFrom > 0 {
		params.Set("primary_release_date.gte", fmt.Sprintf("%d-01-01", f.YearFrom))
	}
	if f.YearTo > 0 {
		params.Set("primary_release_date.lte", fmt.Sprintf("%d-12-31", f.YearTo))
	}
	if f.RatingMin > 0 {
		params.Set("vote_average.gte", strconv.FormatFloat(f.RatingMin, 'f', -1, 64))
	}
	if f.VotesMin > 0 {
		params.Set("vote_count.gte", strconv.Itoa(f.VotesMin))
	}
}

var relativeDateRe = regexp.MustCompile(`^now(?:-(\d+)d)?$`)

// resolveRelativeDate expands preset params like "now-90d" to YYYY-MM-DD
// at request time.
func resolveRelativeDate(v string, now time.Time) string {
	m := relativeDateRe.FindStringSubmatch(v)
	if m == nil {
		return v
	}
	t := now
	if m[1] != "" {
		days, _ := strconv.Atoi(m[1])
		t = t.AddDate(0, 0, -days)
	}
	return t.Format("2006-01-02")
}
