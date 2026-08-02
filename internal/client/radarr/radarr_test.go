package radarr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type capturedAdd struct {
	payload map[string]any
	apiKey  string
}

type addResponse struct {
	status int
	body   any
}

func created() addResponse { return addResponse{status: http.StatusCreated} }

func testServer(t *testing.T, add addResponse, captured *capturedAdd) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}

	mux.HandleFunc("GET /api/v3/qualityprofile", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]any{
			{"id": 4, "name": "HD-1080p"},
			{"id": 5, "name": "Ultra-HD"},
		})
	})
	mux.HandleFunc("GET /api/v3/rootfolder", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]any{{"path": "/movies"}, {"path": "/movies4k"}})
	})
	mux.HandleFunc("GET /api/v3/movie/lookup/tmdb", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tmdbId") != "550" {
			writeJSON(w, map[string]any{})
			return
		}
		writeJSON(w, map[string]any{"title": "Fight Club", "tmdbId": 550, "year": 1999})
	})
	mux.HandleFunc("POST /api/v3/movie", func(w http.ResponseWriter, r *http.Request) {
		captured.apiKey = r.Header.Get("X-Api-Key")
		_ = json.NewDecoder(r.Body).Decode(&captured.payload)
		w.WriteHeader(add.status)
		if add.body != nil {
			_ = json.NewEncoder(w).Encode(add.body)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestProfilesDefaultToTheFirst(t *testing.T) {
	server := testServer(t, created(), &capturedAdd{})
	client := New(server.URL, "secret", "")

	profiles, err := client.Profiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if profiles.DefaultID != 4 || len(profiles.Profiles) != 2 {
		t.Fatalf("profiles = %+v, want both profiles with the first as default", profiles)
	}
}

func TestRequestWithoutProfileUsesTheFirst(t *testing.T) {
	captured := &capturedAdd{}
	server := testServer(t, created(), captured)
	client := New(server.URL, "secret", "")

	if err := client.Request(context.Background(), 550, 0); err != nil {
		t.Fatal(err)
	}
	if captured.apiKey != "secret" {
		t.Fatalf("api key = %q", captured.apiKey)
	}
	payload := captured.payload
	if payload["tmdbId"] != float64(550) || payload["title"] != "Fight Club" {
		t.Fatalf("payload = %v, want the looked-up movie posted back", payload)
	}
	if payload["qualityProfileId"] != float64(4) {
		t.Fatalf("qualityProfileId = %v, want the first profile", payload["qualityProfileId"])
	}
	if payload["rootFolderPath"] != "/movies" {
		t.Fatalf("rootFolderPath = %v, want the first root folder", payload["rootFolderPath"])
	}
	if payload["monitored"] != true {
		t.Fatal("the movie must be added monitored")
	}
	options, _ := payload["addOptions"].(map[string]any)
	if options["searchForMovie"] != true {
		t.Fatal("the add must trigger a search")
	}
}

func TestRequestWithProfile(t *testing.T) {
	captured := &capturedAdd{}
	server := testServer(t, created(), captured)
	client := New(server.URL, "secret", "")

	if err := client.Request(context.Background(), 550, 5); err != nil {
		t.Fatal(err)
	}
	if captured.payload["qualityProfileId"] != float64(5) {
		t.Fatalf("payload = %v", captured.payload)
	}
}

func TestRequestUnknownProfile(t *testing.T) {
	captured := &capturedAdd{}
	server := testServer(t, created(), captured)
	client := New(server.URL, "secret", "")

	if err := client.Request(context.Background(), 550, 42); err == nil {
		t.Fatal("an unknown profile id must fail before anything is sent")
	}
	if captured.payload != nil {
		t.Fatal("no request must go out for an unknown profile")
	}
}

func TestRequestConfiguredRootFolder(t *testing.T) {
	captured := &capturedAdd{}
	server := testServer(t, created(), captured)
	client := New(server.URL, "secret", "/movies4k/")

	if err := client.Request(context.Background(), 550, 0); err != nil {
		t.Fatal(err)
	}
	if captured.payload["rootFolderPath"] != "/movies4k" {
		t.Fatalf("rootFolderPath = %v, want the configured folder as Radarr reports it", captured.payload["rootFolderPath"])
	}
}

func TestRequestUnknownRootFolder(t *testing.T) {
	captured := &capturedAdd{}
	server := testServer(t, created(), captured)
	client := New(server.URL, "secret", "/nope")

	if err := client.Request(context.Background(), 550, 0); err == nil {
		t.Fatal("a root folder Radarr does not know must fail")
	}
	if captured.payload != nil {
		t.Fatal("no request must go out for an unknown root folder")
	}
}

func TestRequestAlreadyAddedCountsAsSuccess(t *testing.T) {
	exists := addResponse{status: http.StatusBadRequest, body: []map[string]any{
		{"errorCode": "MovieExistsValidator", "errorMessage": "This movie has already been added"},
	}}
	server := testServer(t, exists, &capturedAdd{})
	client := New(server.URL, "secret", "")

	if err := client.Request(context.Background(), 550, 0); err != nil {
		t.Fatal(err)
	}
}

func TestRequestOtherValidationFails(t *testing.T) {
	invalid := addResponse{status: http.StatusBadRequest, body: []map[string]any{
		{"errorCode": "PathValidator", "errorMessage": "Path is invalid"},
	}}
	server := testServer(t, invalid, &capturedAdd{})
	client := New(server.URL, "secret", "")

	if err := client.Request(context.Background(), 550, 0); err == nil {
		t.Fatal("a real validation failure must surface")
	}
}

func TestRequestMovieWithoutRecord(t *testing.T) {
	captured := &capturedAdd{}
	server := testServer(t, created(), captured)
	client := New(server.URL, "secret", "")

	if err := client.Request(context.Background(), 1, 0); err == nil {
		t.Fatal("an empty lookup must fail")
	}
	if captured.payload != nil {
		t.Fatal("no add must go out without a lookup result")
	}
}
