package tmdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Jagerente/matcharr/internal/cache"
)

// Config accepts a mirror in BaseURL, some ISPs block TMDB's CloudFront
// ranges; the standard HTTPS_PROXY/HTTP_PROXY vars are honored by the
// transport.
type Config struct {
	APIKey    string
	BaseURL   string
	Language  string
	CacheTTL  time.Duration
	Transport http.RoundTripper
}

type Client struct {
	cfg     Config
	http    *http.Client
	pages   *cache.TTL[pagedMovies]
	genres  *cache.TTL[map[int64]string]
	details *cache.TTL[detailsDTO]
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		http:    &http.Client{Timeout: 15 * time.Second, Transport: cfg.Transport},
		pages:   cache.NewTTL[pagedMovies](cfg.CacheTTL),
		genres:  cache.NewTTL[map[int64]string](cfg.CacheTTL),
		details: cache.NewTTL[detailsDTO](cfg.CacheTTL),
	}
}

type Cache interface {
	Hits() uint64
	Misses() uint64
}

func (c *Client) Caches() map[string]Cache {
	return map[string]Cache{
		"pages":   c.pages,
		"genres":  c.genres,
		"details": c.details,
	}
}

type pagedMovies struct {
	Page       int        `json:"page"`
	TotalPages int        `json:"total_pages"`
	Results    []movieDTO `json:"results"`
}

type movieDTO struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	ReleaseDate   string  `json:"release_date"`
	GenreIDs      []int64 `json:"genre_ids"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
}

// detailsDTO mirrors /movie/{id}, the only endpoint that returns runtime.
type detailsDTO struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	ReleaseDate   string  `json:"release_date"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

func (c *Client) moviesPage(ctx context.Context, path string, params url.Values, page int) (pagedMovies, error) {
	p := url.Values{}
	maps.Copy(p, params)
	p.Set("page", strconv.Itoa(page))

	key := path + "?" + p.Encode()
	if cached, ok := c.pages.Get(key); ok {
		return cached, nil
	}

	var resp pagedMovies
	if err := c.get(ctx, path, p, &resp); err != nil {
		return pagedMovies{}, err
	}
	c.pages.Set(key, resp)
	return resp, nil
}

func (c *Client) movieDetails(ctx context.Context, id int64) (detailsDTO, error) {
	key := strconv.FormatInt(id, 10)
	if cached, ok := c.details.Get(key); ok {
		return cached, nil
	}

	var resp detailsDTO
	if err := c.get(ctx, "/movie/"+key, url.Values{}, &resp); err != nil {
		return detailsDTO{}, err
	}
	c.details.Set(key, resp)
	return resp, nil
}

func (c *Client) movieGenres(ctx context.Context) (map[int64]string, error) {
	if cached, ok := c.genres.Get("movie"); ok {
		return cached, nil
	}

	var resp struct {
		Genres []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
	}
	if err := c.get(ctx, "/genre/movie/list", url.Values{}, &resp); err != nil {
		return nil, err
	}

	out := make(map[int64]string, len(resp.Genres))
	for _, g := range resp.Genres {
		out[g.ID] = g.Name
	}
	c.genres.Set("movie", out)
	return out, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	q := url.Values{}
	maps.Copy(q, params)
	q.Set("api_key", c.cfg.APIKey)
	q.Set("language", c.cfg.Language)

	u := strings.TrimRight(c.cfg.BaseURL, "/") + "/3" + path + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("tmdb: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("tmdb: %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("tmdb: %s: status %d: %s", path, resp.StatusCode, bytes.TrimSpace(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("tmdb: %s: decoding response: %w", path, err)
	}
	return nil
}
