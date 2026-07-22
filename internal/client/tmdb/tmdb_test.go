package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/deck"
)

func testServer(t *testing.T, hits *atomic.Int64, lastQuery *url.Values) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}

	mux.HandleFunc("/3/genre/movie/list", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"genres": []map[string]any{
			{"id": 18, "name": "Drama"},
			{"id": 35, "name": "Comedy"},
		}})
	})
	mux.HandleFunc("/3/discover/movie", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		*lastQuery = r.URL.Query()
		writeJSON(w, map[string]any{
			"page": 1, "total_pages": 3,
			"results": []map[string]any{
				{"id": 1, "title": "Heat", "release_date": "1995-12-15", "genre_ids": []int64{18}, "vote_average": 8.3, "vote_count": 7000},
			},
		})
	})
	mux.HandleFunc("/3/trending/movie/day", func(w http.ResponseWriter, r *http.Request) {
		*lastQuery = r.URL.Query()
		writeJSON(w, map[string]any{
			"page": 1, "total_pages": 1,
			"results": []map[string]any{
				{"id": 1, "title": "Drama Hit", "release_date": "2024-01-01", "genre_ids": []int64{18}},
				{"id": 2, "title": "Comedy Hit", "release_date": "2024-01-01", "genre_ids": []int64{35}},
			},
		})
	})
	mux.HandleFunc("/3/movie/550", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"id": 550, "title": "Fight Club", "release_date": "1999-10-15", "runtime": 139,
			"genres": []map[string]any{{"name": "Drama"}},
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func testSource(t *testing.T, hits *atomic.Int64, lastQuery *url.Values) *Source {
	t.Helper()
	server := testServer(t, hits, lastQuery)
	client := NewClient(Config{APIKey: "k", BaseURL: server.URL, Language: "en-US", CacheTTL: time.Minute})
	source := NewSource(client)
	source.now = func() time.Time { return time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) }
	return source
}

func TestDiscoverAppliesFilters(t *testing.T) {
	var hits atomic.Int64
	var query url.Values
	source := testSource(t, &hits, &query)

	page, err := source.FetchPage(context.Background(), deck.Query{
		Endpoint: EndpointDiscover,
		Filters: domain.Filters{
			GenreIDs:  []int64{18, 35},
			YearFrom:  1990,
			YearTo:    2000,
			RatingMin: 7.5,
			VotesMin:  1000,
		},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"with_genres":              "18|35",
		"primary_release_date.gte": "1990-01-01",
		"primary_release_date.lte": "2000-12-31",
		"vote_average.gte":         "7.5",
		"vote_count.gte":           "1000",
		"api_key":                  "k",
		"language":                 "en-US",
		"page":                     "1",
	}
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Fatalf("query %s = %q, want %q", key, got, value)
		}
	}
	if len(page.Movies) != 1 || page.Movies[0].Year != 1995 || page.Movies[0].Genres[0] != "Drama" {
		t.Fatalf("page = %+v", page.Movies)
	}
	if page.TotalPages != 3 {
		t.Fatalf("TotalPages = %d", page.TotalPages)
	}
}

func TestRelativeDatesResolve(t *testing.T) {
	var hits atomic.Int64
	var query url.Values
	source := testSource(t, &hits, &query)

	_, err := source.FetchPage(context.Background(), deck.Query{
		Endpoint: EndpointDiscover,
		Params:   map[string]string{"primary_release_date.gte": "now-90d", "primary_release_date.lte": "now"},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := query.Get("primary_release_date.gte"); got != "2026-05-03" {
		t.Fatalf("now-90d resolved to %q", got)
	}
	if got := query.Get("primary_release_date.lte"); got != "2026-08-01" {
		t.Fatalf("now resolved to %q", got)
	}
}

func TestListEndpointsFilterLocally(t *testing.T) {
	var hits atomic.Int64
	var query url.Values
	source := testSource(t, &hits, &query)

	page, err := source.FetchPage(context.Background(), deck.Query{
		Endpoint: EndpointTrending,
		Params:   map[string]string{"window": "day"},
		Filters:  domain.Filters{GenreIDs: []int64{35}},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Movies) != 1 || page.Movies[0].Title != "Comedy Hit" {
		t.Fatalf("filtered page = %+v", page.Movies)
	}
	if query.Get("window") != "" {
		t.Fatal("the window pseudo-param must not reach the API")
	}
}

func TestPagesAreCached(t *testing.T) {
	var hits atomic.Int64
	var query url.Values
	source := testSource(t, &hits, &query)

	for range 2 {
		if _, err := source.FetchPage(context.Background(), deck.Query{Endpoint: EndpointDiscover}, 1); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("upstream hits = %d, want the second page served from cache", hits.Load())
	}
}

func TestDetails(t *testing.T) {
	var hits atomic.Int64
	var query url.Values
	source := testSource(t, &hits, &query)

	movie, err := source.Details(context.Background(), 550)
	if err != nil {
		t.Fatal(err)
	}
	if movie.Title != "Fight Club" || movie.Runtime != 139 || movie.Year != 1999 || movie.Genres[0] != "Drama" {
		t.Fatalf("details = %+v", movie)
	}
}

func TestGenresSorted(t *testing.T) {
	var hits atomic.Int64
	var query url.Values
	source := testSource(t, &hits, &query)

	genres, err := source.Genres(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(genres) != 2 || genres[0].Name != "Comedy" || genres[1].Name != "Drama" {
		t.Fatalf("genres = %+v, want alphabetical order", genres)
	}
}

func TestUnknownEndpoint(t *testing.T) {
	var hits atomic.Int64
	var query url.Values
	source := testSource(t, &hits, &query)

	if _, err := source.FetchPage(context.Background(), deck.Query{Endpoint: "bogus"}, 1); err == nil {
		t.Fatal("an unknown endpoint must fail")
	}
}
