package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"

	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/poster"
	"github.com/Jagerente/matcharr/internal/service/catalog"
	"github.com/Jagerente/matcharr/internal/service/deck"
	"github.com/Jagerente/matcharr/internal/service/request"
	"github.com/Jagerente/matcharr/internal/service/session"
	transport "github.com/Jagerente/matcharr/internal/transport/http"
	"github.com/Jagerente/matcharr/internal/transport/http/api"
	"github.com/Jagerente/matcharr/internal/transport/http/ws"
)

type fakeBuilder struct{}

func (fakeBuilder) HasPreset(id string) bool { return id == "p" }

func (fakeBuilder) Build(_ context.Context, req deck.Request) (deck.Result, error) {
	return deck.Result{
		Movies:   []domain.Movie{{ID: 1, Title: "First"}, {ID: 2, Title: "Second"}},
		NextPage: req.StartPage + 1,
	}, nil
}

type fakePresets struct{}

func (fakePresets) List() []domain.Preset {
	return []domain.Preset{{ID: "p", Title: map[string]string{"en": "Preset"}, MediaType: domain.MediaTypeMovie}}
}

type fakeMovies struct{}

func (fakeMovies) Genres(context.Context) ([]domain.Genre, error) {
	return []domain.Genre{{ID: 18, Name: "Drama"}}, nil
}

func (fakeMovies) Details(_ context.Context, id int64) (domain.Movie, error) {
	return domain.Movie{ID: id, Title: "Details", Runtime: 120}, nil
}

type fakeRequester struct{}

func (fakeRequester) Profiles(context.Context) (domain.QualityProfiles, error) {
	return domain.QualityProfiles{Profiles: []domain.QualityProfile{{ID: 4, Name: "HD"}}, DefaultID: 4}, nil
}

func (fakeRequester) Request(context.Context, int64, int) error { return nil }

type stubPosters struct{}

func (stubPosters) Open(_, _ string) (*poster.Cached, bool) { return nil, false }

func (stubPosters) Fetch(context.Context, string, string) ([]byte, string, error) {
	return []byte("img"), "image/jpeg", nil
}

type stackOptions struct {
	urlBase         string
	limiters        transport.Limiters
	disableRequests bool
	wsLimits        ws.Limits
}

type stack struct {
	t    *testing.T
	ts   *httptest.Server
	base string
}

func newStack(t *testing.T, opts stackOptions) *stack {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	sessions := session.New(fakeBuilder{}, session.Limits{
		TTL:             24 * time.Hour,
		EmptyGrace:      15 * time.Minute,
		FinishedTTL:     time.Hour,
		MaxParticipants: 8,
		MaxDeck:         200,
	}, logger)

	var requester request.Requester
	if !opts.disableRequests {
		requester = fakeRequester{}
	}
	requests := request.New(sessions, requester, logger)
	hub := ws.NewHub(logger, sessions, opts.wsLimits)
	sessions.Subscribe(hub.HandleEvent)

	static := fstest.MapFS{
		"index.html":    {Data: []byte("<html>spa</html>")},
		"assets/app.js": {Data: []byte("js")},
	}

	server := transport.New(transport.Config{
		URLBase:   opts.urlBase,
		BodyLimit: 8192,
	}, transport.Deps{
		Logger:          logger,
		Sessions:        sessions,
		Catalog:         catalog.New(fakePresets{}, fakeMovies{}, sessions),
		Requests:        requests,
		Posters:         stubPosters{},
		Hub:             hub,
		Static:          static,
		RequestsEnabled: requests.Enabled(),
		Limiters:        opts.limiters,
	})

	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)
	return &stack{t: t, ts: ts, base: opts.urlBase}
}

func (s *stack) do(method, path, token string, body any) *http.Response {
	s.t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, s.ts.URL+s.base+path, reader)
	if err != nil {
		s.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set(api.TokenHeader, token)
	}
	resp, err := s.ts.Client().Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	return resp
}

func (s *stack) json(method, path, token string, body any, wantStatus int) map[string]any {
	s.t.Helper()
	resp := s.do(method, path, token, body)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		s.t.Fatalf("%s %s = %d, want %d (body %s)", method, path, resp.StatusCode, wantStatus, raw)
	}
	if len(raw) == 0 {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		s.t.Fatalf("%s %s returned unparseable JSON: %v (%s)", method, path, err, raw)
	}
	return decoded
}

type creds struct {
	code  string
	token string
	id    string
}

func decodeCreds(t *testing.T, body map[string]any) creds {
	t.Helper()
	you := body["you"].(map[string]any)
	return creds{
		code:  body["code"].(string),
		token: body["token"].(string),
		id:    you["id"].(string),
	}
}

func createSession(t *testing.T, s *stack, name string) creds {
	t.Helper()
	body := s.json(http.MethodPost, "/api/sessions", "", map[string]any{
		"name":     name,
		"settings": map[string]any{"presetId": "p"},
	}, http.StatusCreated)
	return decodeCreds(t, body)
}

func joinSession(t *testing.T, s *stack, code, name string) creds {
	t.Helper()
	body := s.json(http.MethodPost, "/api/sessions/"+code+"/join", "", map[string]any{"name": name}, http.StatusOK)
	return decodeCreds(t, body)
}

func TestFullSessionFlow(t *testing.T) {
	s := newStack(t, stackOptions{})

	host := createSession(t, s, "Host")
	guest := joinSession(t, s, host.code, "Guest")

	s.json(http.MethodPost, "/api/sessions/"+host.code+"/start", host.token, nil, http.StatusNoContent)

	deckBody := s.json(http.MethodGet, "/api/sessions/"+host.code+"/deck", host.token, nil, http.StatusOK)
	movies := deckBody["movies"].([]any)
	if len(movies) != 2 {
		t.Fatalf("deck = %v", movies)
	}

	outOfOrder := s.json(http.MethodPost, "/api/sessions/"+host.code+"/swipe", host.token,
		map[string]any{"movieId": 2, "verdict": "like"}, http.StatusBadRequest)
	if outOfOrder["reason"] != "swipe_out_of_order" {
		t.Fatalf("error envelope = %v", outOfOrder)
	}

	swipe := map[string]any{"movieId": 1, "verdict": "like"}
	s.json(http.MethodPost, "/api/sessions/"+host.code+"/swipe", host.token, swipe, http.StatusNoContent)
	s.json(http.MethodPost, "/api/sessions/"+host.code+"/swipe", guest.token, swipe, http.StatusNoContent)

	state := s.json(http.MethodGet, "/api/sessions/"+host.code, host.token, nil, http.StatusOK)
	inner := state["state"].(map[string]any)
	if inner["status"] != "finished" || len(inner["matches"].([]any)) != 1 {
		t.Fatalf("state after the match = %v", inner)
	}
	if inner["results"] == nil {
		t.Fatal("a finished session must carry results")
	}
}

func TestErrorEnvelopes(t *testing.T) {
	s := newStack(t, stackOptions{})
	host := createSession(t, s, "Host")

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		body       any
		wantStatus int
		wantReason string
	}{
		{"unknown session", http.MethodGet, "/api/sessions/XXXX", "t", nil, http.StatusNotFound, "session_not_found"},
		{"bad token", http.MethodGet, "/api/sessions/" + host.code, "wrong", nil, http.StatusUnauthorized, "invalid_token"},
		{"unknown endpoint", http.MethodGet, "/api/bogus", "", nil, http.StatusNotFound, "bad_request"},
		{"empty name", http.MethodPost, "/api/sessions", "", map[string]any{"settings": map[string]any{"presetId": "p"}}, http.StatusBadRequest, "name_required"},
		{"unknown preset", http.MethodPost, "/api/sessions", "", map[string]any{"name": "A", "settings": map[string]any{"presetId": "zzz"}}, http.StatusBadRequest, "unknown_preset"},
		{"validator catches a bad verdict", http.MethodPost, "/api/sessions/" + host.code + "/swipe", host.token, map[string]any{"movieId": 1, "verdict": "maybe"}, http.StatusBadRequest, "bad_request"},
		{"invalid movie id", http.MethodGet, "/api/sessions/" + host.code + "/movies/abc", host.token, nil, http.StatusBadRequest, "bad_request"},
		{"guest cannot start", http.MethodPost, "/api/sessions/" + host.code + "/start", "nope", nil, http.StatusUnauthorized, "invalid_token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := s.json(tt.method, tt.path, tt.token, tt.body, tt.wantStatus)
			if body["reason"] != tt.wantReason {
				t.Fatalf("envelope = %v, want reason %q", body, tt.wantReason)
			}
			if _, ok := body["message"].(string); !ok {
				t.Fatalf("envelope %v lacks a message", body)
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	s := newStack(t, stackOptions{})
	req, _ := http.NewRequest(http.MethodPost, s.ts.URL+"/api/sessions", strings.NewReader("{broken"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed JSON = %d", resp.StatusCode)
	}
}

func TestCatalogEndpoints(t *testing.T) {
	s := newStack(t, stackOptions{})

	config := s.json(http.MethodGet, "/api/config", "", nil, http.StatusOK)
	if config["requestsEnabled"] != true {
		t.Fatalf("config = %v", config)
	}

	resp := s.do(http.MethodGet, "/api/genres", "", nil)
	defer resp.Body.Close()
	var genres []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&genres); err != nil || len(genres) != 1 || genres[0]["name"] != "Drama" {
		t.Fatalf("genres = %v, %v", genres, err)
	}

	presetsResp := s.do(http.MethodGet, "/api/presets", "", nil)
	defer presetsResp.Body.Close()
	var presets []map[string]any
	if err := json.NewDecoder(presetsResp.Body).Decode(&presets); err != nil || len(presets) != 1 {
		t.Fatalf("presets = %v, %v", presets, err)
	}
}

func TestMovieDetailsAuthorization(t *testing.T) {
	s := newStack(t, stackOptions{})
	host := createSession(t, s, "Host")
	joinSession(t, s, host.code, "Guest")
	s.json(http.MethodPost, "/api/sessions/"+host.code+"/start", host.token, nil, http.StatusNoContent)

	details := s.json(http.MethodGet, "/api/sessions/"+host.code+"/movies/1", host.token, nil, http.StatusOK)
	if details["title"] != "Details" {
		t.Fatalf("details = %v", details)
	}

	outside := s.json(http.MethodGet, "/api/sessions/"+host.code+"/movies/999", host.token, nil, http.StatusBadRequest)
	if outside["reason"] != "unknown_movie" {
		t.Fatalf("envelope = %v, the catalog must not be browsable", outside)
	}
}

func TestRequestFlow(t *testing.T) {
	s := newStack(t, stackOptions{})
	host := createSession(t, s, "Host")
	guest := joinSession(t, s, host.code, "Guest")
	s.json(http.MethodPost, "/api/sessions/"+host.code+"/start", host.token, nil, http.StatusNoContent)

	profiles := s.json(http.MethodGet, "/api/sessions/"+host.code+"/request/profiles", host.token, nil, http.StatusOK)
	if profiles["defaultId"] != float64(4) {
		t.Fatalf("profiles = %v", profiles)
	}

	forbidden := s.json(http.MethodGet, "/api/sessions/"+host.code+"/request/profiles", guest.token, nil, http.StatusForbidden)
	if forbidden["reason"] != "host_only" {
		t.Fatalf("envelope = %v", forbidden)
	}

	s.json(http.MethodPost, "/api/sessions/"+host.code+"/request", host.token,
		map[string]any{"movieId": 1, "profileId": 4}, http.StatusNoContent)

	state := s.json(http.MethodGet, "/api/sessions/"+host.code, host.token, nil, http.StatusOK)
	requested := state["state"].(map[string]any)["requested"].([]any)
	if len(requested) != 1 || requested[0] != float64(1) {
		t.Fatalf("requested = %v", requested)
	}
}

func TestRequestsDisabled(t *testing.T) {
	s := newStack(t, stackOptions{disableRequests: true})
	host := createSession(t, s, "Host")

	config := s.json(http.MethodGet, "/api/config", "", nil, http.StatusOK)
	if config["requestsEnabled"] != false {
		t.Fatalf("config = %v", config)
	}
	disabled := s.json(http.MethodGet, "/api/sessions/"+host.code+"/request/profiles", host.token, nil, http.StatusNotImplemented)
	if disabled["reason"] != "requests_disabled" {
		t.Fatalf("envelope = %v", disabled)
	}
}

func TestSPA(t *testing.T) {
	s := newStack(t, stackOptions{})

	tests := []struct {
		path         string
		wantBody     string
		wantCache    string
		wantContains bool
	}{
		{"/", "<html>spa</html>", "no-cache", true},
		{"/join/ABCD", "<html>spa</html>", "no-cache", true},
		{"/assets/app.js", "js", "public, max-age=31536000, immutable", true},
	}
	for _, tt := range tests {
		resp := s.do(http.MethodGet, tt.path, "", nil)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != tt.wantBody {
			t.Fatalf("GET %s = %d %q", tt.path, resp.StatusCode, body)
		}
		if got := resp.Header.Get("Cache-Control"); got != tt.wantCache {
			t.Fatalf("GET %s Cache-Control = %q, want %q", tt.path, got, tt.wantCache)
		}
	}
}

func TestURLBaseMounting(t *testing.T) {
	s := newStack(t, stackOptions{urlBase: "/matcharr"})

	resp, err := s.ts.Client().Get(s.ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d, must stay at the root", resp.StatusCode)
	}

	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	bare, err := noRedirect.Get(s.ts.URL + "/matcharr")
	if err != nil {
		t.Fatal(err)
	}
	bare.Body.Close()
	if bare.StatusCode != http.StatusPermanentRedirect {
		t.Fatalf("bare base = %d, want 308", bare.StatusCode)
	}

	root, err := noRedirect.Get(s.ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	root.Body.Close()
	if root.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("root = %d, want 307 to the base", root.StatusCode)
	}

	config := s.json(http.MethodGet, "/api/config", "", nil, http.StatusOK)
	if _, ok := config["requestsEnabled"]; !ok {
		t.Fatalf("config under the base = %v", config)
	}
}

type stubLimiter struct{ allowed int }

func (l *stubLimiter) Allow(string) bool {
	l.allowed--
	return l.allowed >= 0
}

func TestRateLimitedRoute(t *testing.T) {
	s := newStack(t, stackOptions{limiters: transport.Limiters{Catalog: &stubLimiter{allowed: 1}}})

	s.json(http.MethodGet, "/api/config", "", nil, http.StatusOK)
	throttled := s.json(http.MethodGet, "/api/config", "", nil, http.StatusTooManyRequests)
	if throttled["reason"] != "rate_limited" {
		t.Fatalf("envelope = %v", throttled)
	}
}

func TestPosterRoute(t *testing.T) {
	s := newStack(t, stackOptions{})

	resp := s.do(http.MethodGet, "/api/image/w185/abc.jpg", "", nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "img" {
		t.Fatalf("poster = %d %q", resp.StatusCode, body)
	}

	bad := s.json(http.MethodGet, "/api/image/w999/abc.jpg", "", nil, http.StatusBadRequest)
	if bad["reason"] != "bad_request" {
		t.Fatalf("envelope = %v", bad)
	}
	traversal := s.json(http.MethodGet, "/api/image/w185/..%2Fsecret.jpg", "", nil, http.StatusBadRequest)
	if traversal["reason"] != "bad_request" {
		t.Fatalf("envelope = %v", traversal)
	}
}

func wsURL(ts *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(ts.URL, "http") + path
}

func readMessage(t *testing.T, ctx context.Context, conn *websocket.Conn) map[string]any {
	t.Helper()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatal(err)
	}
	return msg
}

func TestWebSocket(t *testing.T) {
	s := newStack(t, stackOptions{})
	host := createSession(t, s, "Host")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(s.ts, "/api/sessions/"+host.code+"/ws?token="+host.token), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Connecting flips the presence first, so a lobby_update may arrive
	// ahead of the snapshot; both carry the full state.
	snapshot := readMessage(t, ctx, conn)
	for range 3 {
		if snapshot["type"] == "snapshot" {
			break
		}
		snapshot = readMessage(t, ctx, conn)
	}
	if snapshot["type"] != "snapshot" {
		t.Fatalf("no snapshot arrived, last message = %v", snapshot)
	}
	you := snapshot["you"].(map[string]any)
	if you["id"] != host.id || you["isHost"] != true {
		t.Fatalf("snapshot you = %v", you)
	}

	// The socket carries presence: the host shows online in its own snapshot
	// state or in the follow-up lobby update.
	joinSession(t, s, host.code, "Guest")
	deadline := time.Now().Add(2 * time.Second)
	for {
		msg := readMessage(t, ctx, conn)
		if msg["type"] == "lobby_update" {
			state := msg["state"].(map[string]any)
			if len(state["participants"].([]any)) == 2 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("no lobby update with the guest arrived")
		}
	}
}

func TestWebSocketConnectionCap(t *testing.T) {
	s := newStack(t, stackOptions{wsLimits: ws.Limits{ConnsPerClient: 1}})
	host := createSession(t, s, "Host")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	target := wsURL(s.ts, "/api/sessions/"+host.code+"/ws?token="+host.token)
	conn, _, err := websocket.Dial(ctx, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, resp, err := websocket.Dial(ctx, target, nil)
	if err == nil {
		t.Fatal("the second socket must be rejected")
	}
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second dial status = %v", resp)
	}
}

func TestWebSocketRejectsBadToken(t *testing.T) {
	s := newStack(t, stackOptions{})
	host := createSession(t, s, "Host")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, resp, err := websocket.Dial(ctx, wsURL(s.ts, "/api/sessions/"+host.code+"/ws?token=wrong"), nil)
	if err == nil {
		t.Fatal("a bad token must not open a socket")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("dial status = %v", resp)
	}
}

func TestHealthz(t *testing.T) {
	s := newStack(t, stackOptions{})
	resp, err := s.ts.Client().Get(s.ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("healthz = %d %q", resp.StatusCode, body)
	}
}
