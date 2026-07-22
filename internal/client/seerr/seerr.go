package seerr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Jagerente/matcharr/internal/cache"
	"github.com/Jagerente/matcharr/internal/domain"
)

const (
	httpTimeout    = 15 * time.Second
	radarrCacheKey = "radarr"
	radarrCacheTTL = 10 * time.Minute
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
	radarr  *cache.TTL[radarrTarget]
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: httpTimeout},
		radarr:  cache.NewTTL[radarrTarget](radarrCacheTTL),
	}
}

func (c *Client) Profiles(ctx context.Context) (domain.QualityProfiles, error) {
	target, err := c.target(ctx)
	if err != nil {
		return domain.QualityProfiles{}, err
	}
	return domain.QualityProfiles{Profiles: target.profiles, DefaultID: target.defaultID}, nil
}

func (c *Client) Request(ctx context.Context, movieID int64, profileID int) error {
	payload := map[string]any{
		"mediaType": "movie",
		"mediaId":   movieID,
	}
	// Without an explicit profile Seerr applies the server's own default,
	// so the pair is only sent when the host picked something.
	if profileID != 0 {
		target, err := c.target(ctx)
		if err != nil {
			return err
		}
		if !target.hasProfile(profileID) {
			return fmt.Errorf("seerr: unknown quality profile %d", profileID)
		}
		payload["serverId"] = target.serverID
		payload["profileId"] = profileID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/api/v1/request", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("seerr: %w", err)
	}
	defer resp.Body.Close()

	// 409 means already requested, which counts as success for the user.
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict {
		return nil
	}
	return statusError(resp)
}

type radarrTarget struct {
	serverID  int
	defaultID int
	profiles  []domain.QualityProfile
}

func (t radarrTarget) hasProfile(id int) bool {
	return slices.ContainsFunc(t.profiles, func(p domain.QualityProfile) bool { return p.ID == id })
}

type radarrServer struct {
	ID              int  `json:"id"`
	Is4K            bool `json:"is4k"`
	IsDefault       bool `json:"isDefault"`
	ActiveProfileID int  `json:"activeProfileId"`
}

func (c *Client) target(ctx context.Context) (radarrTarget, error) {
	if target, ok := c.radarr.Get(radarrCacheKey); ok {
		return target, nil
	}

	var servers []radarrServer
	if err := c.get(ctx, "/api/v1/service/radarr", &servers); err != nil {
		return radarrTarget{}, err
	}
	server, ok := pickServer(servers)
	if !ok {
		return radarrTarget{}, errors.New("seerr: no Radarr server is configured")
	}

	var details struct {
		Profiles []profileDTO `json:"profiles"`
	}
	if err := c.get(ctx, fmt.Sprintf("/api/v1/service/radarr/%d", server.ID), &details); err != nil {
		return radarrTarget{}, err
	}

	profiles := make([]domain.QualityProfile, 0, len(details.Profiles))
	for _, p := range details.Profiles {
		profiles = append(profiles, domain.QualityProfile{ID: p.ID, Name: p.Name})
	}

	target := radarrTarget{
		serverID:  server.ID,
		defaultID: server.ActiveProfileID,
		profiles:  profiles,
	}
	c.radarr.Set(radarrCacheKey, target)
	return target, nil
}

type profileDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// pickServer prefers the non-4K server: a 4K instance is a separate
// Radarr with its own profile ids, and requests go out as non-4K, so both
// the profiles and the server id must come from the same place.
func pickServer(servers []radarrServer) (radarrServer, bool) {
	var fallback radarrServer
	found := false
	for _, s := range servers {
		if s.Is4K {
			continue
		}
		if s.IsDefault {
			return s, true
		}
		if !found {
			fallback, found = s, true
		}
	}
	return fallback, found
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("seerr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return statusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("seerr: decoding %s: %w", path, err)
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
	return fmt.Errorf("seerr: status %d: %s", resp.StatusCode, bytes.TrimSpace(snippet))
}
