package seerr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type capturedRequest struct {
	payload map[string]any
	apiKey  string
}

func testServer(t *testing.T, requestStatus int, captured *capturedRequest) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}

	mux.HandleFunc("GET /api/v1/service/radarr", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]any{
			{"id": 9, "is4k": true, "isDefault": true, "activeProfileId": 99},
			{"id": 1, "is4k": false, "isDefault": true, "activeProfileId": 4},
		})
	})
	mux.HandleFunc("GET /api/v1/service/radarr/1", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"profiles": []map[string]any{
			{"id": 4, "name": "HD-1080p"},
			{"id": 5, "name": "Ultra-HD"},
		}})
	})
	mux.HandleFunc("POST /api/v1/request", func(w http.ResponseWriter, r *http.Request) {
		captured.apiKey = r.Header.Get("X-Api-Key")
		_ = json.NewDecoder(r.Body).Decode(&captured.payload)
		w.WriteHeader(requestStatus)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestProfilesPickTheNon4KServer(t *testing.T) {
	server := testServer(t, http.StatusCreated, &capturedRequest{})
	client := New(server.URL, "secret")

	profiles, err := client.Profiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if profiles.DefaultID != 4 || len(profiles.Profiles) != 2 {
		t.Fatalf("profiles = %+v, want the non-4K server's set", profiles)
	}
}

func TestRequestWithoutProfileOmitsServer(t *testing.T) {
	captured := &capturedRequest{}
	server := testServer(t, http.StatusCreated, captured)
	client := New(server.URL, "secret")

	if err := client.Request(context.Background(), 550, 0); err != nil {
		t.Fatal(err)
	}
	if captured.apiKey != "secret" {
		t.Fatalf("api key = %q", captured.apiKey)
	}
	if captured.payload["mediaId"] != float64(550) {
		t.Fatalf("payload = %v", captured.payload)
	}
	if _, has := captured.payload["profileId"]; has {
		t.Fatal("without an explicit profile the server's default must apply")
	}
}

func TestRequestWithProfile(t *testing.T) {
	captured := &capturedRequest{}
	server := testServer(t, http.StatusCreated, captured)
	client := New(server.URL, "secret")

	if err := client.Request(context.Background(), 550, 5); err != nil {
		t.Fatal(err)
	}
	if captured.payload["profileId"] != float64(5) || captured.payload["serverId"] != float64(1) {
		t.Fatalf("payload = %v", captured.payload)
	}
}

func TestRequestUnknownProfile(t *testing.T) {
	captured := &capturedRequest{}
	server := testServer(t, http.StatusCreated, captured)
	client := New(server.URL, "secret")

	if err := client.Request(context.Background(), 550, 42); err == nil {
		t.Fatal("an unknown profile id must fail before anything is sent")
	}
	if captured.payload != nil {
		t.Fatal("no request must go out for an unknown profile")
	}
}

func TestRequestConflictCountsAsSuccess(t *testing.T) {
	server := testServer(t, http.StatusConflict, &capturedRequest{})
	client := New(server.URL, "secret")

	// 409 means already requested, which is success for the user.
	if err := client.Request(context.Background(), 550, 0); err != nil {
		t.Fatal(err)
	}
}
