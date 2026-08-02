package radarr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Jagerente/matcharr/internal/cache"
	"github.com/Jagerente/matcharr/internal/domain"
)

const (
	httpTimeout    = 15 * time.Second
	targetCacheKey = "target"
	targetCacheTTL = 10 * time.Minute
	// minimumAvailability mirrors Radarr's own default for added movies.
	minimumAvailability = "announced"
)

type Client struct {
	baseURL    string
	apiKey     string
	rootFolder string
	http       *http.Client
	target     *cache.TTL[target]
}

func New(baseURL, apiKey, rootFolder string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		rootFolder: rootFolder,
		http:       &http.Client{Timeout: httpTimeout},
		target:     cache.NewTTL[target](targetCacheTTL),
	}
}

func (c *Client) Profiles(ctx context.Context) (domain.QualityProfiles, error) {
	t, err := c.load(ctx)
	if err != nil {
		return domain.QualityProfiles{}, err
	}
	return domain.QualityProfiles{Profiles: t.profiles, DefaultID: t.defaultID}, nil
}

func (c *Client) Request(ctx context.Context, movieID int64, profileID int) error {
	t, err := c.load(ctx)
	if err != nil {
		return err
	}
	// Radarr requires a profile on every add and has no server-side default
	// the way Seerr does, so a zero id resolves to the first profile here.
	if profileID == 0 {
		profileID = t.defaultID
	} else if !t.hasProfile(profileID) {
		return fmt.Errorf("radarr: unknown quality profile %d", profileID)
	}

	movie, err := c.lookup(ctx, movieID)
	if err != nil {
		return err
	}
	movie["qualityProfileId"] = profileID
	movie["rootFolderPath"] = t.rootFolder
	movie["monitored"] = true
	movie["minimumAvailability"] = minimumAvailability
	movie["addOptions"] = map[string]any{"searchForMovie": true}

	return c.add(ctx, movie)
}

// lookup fetches the full movie resource: Radarr's add endpoint expects the
// looked-up object posted back, not a bare TMDB id.
func (c *Client) lookup(ctx context.Context, movieID int64) (map[string]any, error) {
	var movie map[string]any
	path := "/api/v3/movie/lookup/tmdb?tmdbId=" + strconv.FormatInt(movieID, 10)
	if err := c.get(ctx, path, &movie); err != nil {
		return nil, err
	}
	if movie["title"] == nil {
		return nil, fmt.Errorf("radarr: movie %d has no TMDB record", movieID)
	}
	return movie, nil
}

func (c *Client) add(ctx context.Context, movie map[string]any) error {
	body, err := json.Marshal(movie)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/api/v3/movie", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("radarr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	// A movie already in the library fails validation, which counts as
	// success for the user, like Seerr's 409.
	if resp.StatusCode == http.StatusBadRequest && alreadyAdded(snippet) {
		return nil
	}
	return fmt.Errorf("radarr: status %d: %s", resp.StatusCode, bytes.TrimSpace(snippet))
}

func alreadyAdded(body []byte) bool {
	var failures []struct {
		ErrorCode string `json:"errorCode"`
	}
	if json.Unmarshal(body, &failures) != nil {
		return false
	}
	for _, f := range failures {
		if f.ErrorCode == "MovieExistsValidator" {
			return true
		}
	}
	return false
}

type target struct {
	defaultID  int
	rootFolder string
	profiles   []domain.QualityProfile
}

func (t target) hasProfile(id int) bool {
	return slices.ContainsFunc(t.profiles, func(p domain.QualityProfile) bool { return p.ID == id })
}

func (c *Client) load(ctx context.Context) (target, error) {
	if t, ok := c.target.Get(targetCacheKey); ok {
		return t, nil
	}

	var dtos []profileDTO
	if err := c.get(ctx, "/api/v3/qualityprofile", &dtos); err != nil {
		return target{}, err
	}
	if len(dtos) == 0 {
		return target{}, errors.New("radarr: no quality profiles are configured")
	}
	profiles := make([]domain.QualityProfile, 0, len(dtos))
	for _, p := range dtos {
		profiles = append(profiles, domain.QualityProfile{ID: p.ID, Name: p.Name})
	}

	var folders []rootFolderDTO
	if err := c.get(ctx, "/api/v3/rootfolder", &folders); err != nil {
		return target{}, err
	}
	folder, err := pickRootFolder(folders, c.rootFolder)
	if err != nil {
		return target{}, err
	}

	t := target{defaultID: profiles[0].ID, rootFolder: folder, profiles: profiles}
	c.target.Set(targetCacheKey, t)
	return t, nil
}

type profileDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type rootFolderDTO struct {
	Path string `json:"path"`
}

// pickRootFolder compares trimmed paths: Radarr reports folders without a
// trailing separator, while the configured value may carry one.
func pickRootFolder(folders []rootFolderDTO, configured string) (string, error) {
	if len(folders) == 0 {
		return "", errors.New("radarr: no root folder is configured")
	}
	if configured == "" {
		return folders[0].Path, nil
	}
	want := trimSep(configured)
	for _, f := range folders {
		if trimSep(f.Path) == want {
			return f.Path, nil
		}
	}
	return "", fmt.Errorf("radarr: root folder %q is not configured in Radarr", configured)
}

func trimSep(path string) string { return strings.TrimRight(path, "/\\") }

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("radarr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return statusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("radarr: decoding %s: %w", path, err)
	}
	return nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	return req, nil
}

func statusError(resp *http.Response) error {
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("radarr: status %d: %s", resp.StatusCode, bytes.TrimSpace(snippet))
}
