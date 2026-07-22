package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/transport/http/httperr"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		headerVal  string
		remoteAddr string
		want       string
	}{
		{"socket address by default", "", "", "10.0.0.5:1234", "10.0.0.5"},
		{"trusted header wins", "CF-Connecting-IP", "1.2.3.4", "10.0.0.5:1234", "1.2.3.4"},
		{"forwarded chain takes the first hop", "X-Forwarded-For", "1.2.3.4, 5.6.7.8", "10.0.0.5:1234", "1.2.3.4"},
		{"empty header falls back to the socket", "CF-Connecting-IP", "", "10.0.0.5:1234", "10.0.0.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.headerVal != "" {
				req.Header.Set(tt.header, tt.headerVal)
			}
			if got := (ClientIP{Header: tt.header}).FromRequest(req); got != tt.want {
				t.Fatalf("FromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}

type stubLimiter struct{ allowed int }

func (l *stubLimiter) Allow(string) bool {
	l.allowed--
	return l.allowed >= 0
}

func TestRateLimit(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = httperr.Handle
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		RateLimit(&stubLimiter{allowed: 1}, ClientIP{}))

	first := httptest.NewRecorder()
	e.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request = %d", first.Code)
	}

	second := httptest.NewRecorder()
	e.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request = %d, want 429", second.Code)
	}
	if second.Header().Get("Retry-After") != "5" {
		t.Fatal("a throttled response must carry Retry-After")
	}
}

type stubObserver struct {
	mu       sync.Mutex
	inFlight int
	routes   []string
	statuses []int
	reasons  []string
}

func (o *stubObserver) RequestStarted() { o.mu.Lock(); o.inFlight++; o.mu.Unlock() }
func (o *stubObserver) RequestDone()    { o.mu.Lock(); o.inFlight--; o.mu.Unlock() }

func (o *stubObserver) ObserveRequest(route, _ string, status int, _ time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.routes = append(o.routes, route)
	o.statuses = append(o.statuses, status)
}

func (o *stubObserver) ObserveError(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.reasons = append(o.reasons, reason)
}

func TestObserveRecordsFinalStatusAndReason(t *testing.T) {
	observer := &stubObserver{}
	e := echo.New()
	e.HTTPErrorHandler = httperr.Handle
	e.Use(Observe(ObserveConfig{Logger: slog.New(slog.DiscardHandler), Observer: observer}))
	e.GET("/sessions/:code", func(echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "nope")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions/AAAA", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(observer.routes) != 1 || observer.routes[0] != "/sessions/{code}" {
		t.Fatalf("routes = %v, want the dashboard-style pattern", observer.routes)
	}
	if observer.statuses[0] != http.StatusNotFound {
		t.Fatalf("recorded status = %d, want the one the client received", observer.statuses[0])
	}
	if len(observer.reasons) != 1 || observer.reasons[0] != "bad_request" {
		t.Fatalf("reasons = %v", observer.reasons)
	}
	if observer.inFlight != 0 {
		t.Fatalf("in flight = %d after completion", observer.inFlight)
	}
}

func TestObserveSkipper(t *testing.T) {
	observer := &stubObserver{}
	e := echo.New()
	e.Use(Observe(ObserveConfig{
		Logger:   slog.New(slog.DiscardHandler),
		Observer: observer,
		Skipper:  func(c echo.Context) bool { return c.Path() == "/ws" },
	}))
	e.GET("/ws", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ws", nil))
	if len(observer.routes) != 0 {
		t.Fatalf("routes = %v, want the socket exempt", observer.routes)
	}
}

func TestRouteLabel(t *testing.T) {
	tests := []struct {
		path string
		trim string
		want string
	}{
		{"/api/sessions/:code/join", "", "/api/sessions/{code}/join"},
		{"/matcharr/api/image/:size/:file", "/matcharr", "/api/image/{size}/{file}"},
		{"/*", "", "/"},
		{"/matcharr/*", "/matcharr", "/"},
		{"/healthz", "", "/healthz"},
	}
	for _, tt := range tests {
		if got := routeLabel(tt.path, tt.trim); got != tt.want {
			t.Fatalf("routeLabel(%q, %q) = %q, want %q", tt.path, tt.trim, got, tt.want)
		}
	}
}

func TestBodyLimit(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = httperr.Handle
	e.Use(BodyLimit(8))
	e.POST("/", func(c echo.Context) error {
		var v map[string]any
		if err := c.Bind(&v); err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	send := func(body string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		e.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := send(`{}`); got != http.StatusOK {
		t.Fatalf("small body = %d", got)
	}
	if got := send(`{"name":"far too long for the eight byte budget"}`); got != http.StatusBadRequest {
		t.Fatalf("oversized body = %d, want a bind failure", got)
	}
}
