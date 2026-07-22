package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "matcharr"

func httpDurationBuckets() []float64 {
	return []float64{
		0.0001, 0.00025, 0.0005,
		0.001, 0.0025, 0.005,
		0.01, 0.025, 0.05,
		0.1, 0.25, 0.5,
		1, 2.5, 5, 10, 30,
	}
}

func upstreamDurationBuckets() []float64 {
	return []float64{
		0.05, 0.1, 0.25, 0.5,
		1, 2, 3, 5, 7.5, 10, 15,
	}
}

// Snapshot is deliberately label-typed as plain strings: metrics describe
// the instance and must not pull business packages into this one.
type Snapshot struct {
	Sessions     map[string]int
	Participants int
	Connections  int
}

type Source func() Snapshot

type Metrics struct {
	registry *prometheus.Registry

	requests   *prometheus.CounterVec
	duration   *prometheus.HistogramVec
	inFlight   prometheus.Gauge
	apiErrors  *prometheus.CounterVec
	sessionEvs *prometheus.CounterVec

	tmdbRequests *prometheus.CounterVec
	tmdbDuration *prometheus.HistogramVec

	posters      *prometheus.CounterVec
	posterBytes  prometheus.Counter
	posterEvicts prometheus.Counter
	posterCache  prometheus.Gauge
}

func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "HTTP requests by route, method and status (route is the pattern, never the concrete path, so session codes stay out of the labels).",
		}, []string{"route", "method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration by route (the event socket is excluded, it lives for a whole session).",
			Buckets:   httpDurationBuckets(),
		}, []string{"route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "HTTP requests being served right now.",
		}),
		apiErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_errors_total",
			Help:      "API failures by reason, the same reason the client renders.",
		}, []string{"reason"}),
		sessionEvs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "session_events_total",
			Help:      "Session engine events by type.",
		}, []string{"event"}),
		tmdbRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tmdb_requests_total",
			Help:      "Calls that actually left for the movie source, by endpoint group and status (status is the HTTP code, or error when the round trip itself failed).",
		}, []string{"endpoint", "status"}),
		tmdbDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "tmdb_request_duration_seconds",
			Help:      "Movie source round trip by endpoint group (cache hits never get here).",
			Buckets:   upstreamDurationBuckets(),
		}, []string{"endpoint"}),
		posters: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "posters_total",
			Help:      "Poster requests by how they were served.",
		}, []string{"result"}),
		posterBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "poster_bytes_total",
			Help:      "Poster bytes written to clients, the uplink cost of the deck.",
		}),
		posterEvicts: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "poster_cache_evicted_total",
			Help:      "Cached posters dropped to stay inside the cache budget.",
		}),
		posterCache: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "poster_cache_bytes",
			Help:      "Poster cache size as of the last sweep.",
		}),
	}

	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.requests, m.duration, m.inFlight, m.apiErrors, m.sessionEvs,
		m.tmdbRequests, m.tmdbDuration,
		m.posters, m.posterBytes, m.posterEvicts, m.posterCache,
	)
	return m
}

func (m *Metrics) SetSource(src Source) {
	m.registry.MustRegister(snapshotCollector{
		src:          src,
		sessions:     desc("sessions", "Sessions currently held in memory, by lifecycle status.", "status"),
		participants: desc("participants", "People across all live sessions."),
		connections:  desc("ws_connections", "Open event sockets."),
	})
}

type snapshotCollector struct {
	src                                 Source
	sessions, participants, connections *prometheus.Desc
}

func (c snapshotCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.sessions
	ch <- c.participants
	ch <- c.connections
}

func (c snapshotCollector) Collect(ch chan<- prometheus.Metric) {
	snapshot := c.src()
	for status, count := range snapshot.Sessions {
		ch <- prometheus.MustNewConstMetric(c.sessions, prometheus.GaugeValue, float64(count), status)
	}
	ch <- prometheus.MustNewConstMetric(c.participants, prometheus.GaugeValue, float64(snapshot.Participants))
	ch <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(snapshot.Connections))
}

func desc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "", name), help, labels, nil)
}

func (m *Metrics) TrackCache(name string, hits, misses func() uint64) {
	m.registry.MustRegister(
		counterFunc("cache_hits_total", "Cache hits.", name, func() float64 { return float64(hits()) }),
		counterFunc("cache_misses_total", "Cache misses.", name, func() float64 { return float64(misses()) }),
	)
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{Registry: m.registry})
}

func (m *Metrics) ObserveRequest(route, method string, status int, d time.Duration) {
	m.requests.WithLabelValues(route, method, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(route).Observe(d.Seconds())
}

func (m *Metrics) RequestStarted() { m.inFlight.Inc() }
func (m *Metrics) RequestDone()    { m.inFlight.Dec() }

func (m *Metrics) ObserveError(reason string) {
	m.apiErrors.WithLabelValues(reason).Inc()
}

func (m *Metrics) ObserveSessionEvent(event string) {
	m.sessionEvs.WithLabelValues(event).Inc()
}

func (m *Metrics) ObservePoster(result string, bytes int) {
	m.posters.WithLabelValues(result).Inc()
	m.posterBytes.Add(float64(bytes))
}

func (m *Metrics) ObservePosterCache(size int64, evicted int) {
	m.posterCache.Set(float64(size))
	m.posterEvicts.Add(float64(evicted))
}

func (m *Metrics) Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return roundTripper{base: base, m: m}
}

type roundTripper struct {
	base http.RoundTripper
	m    *Metrics
}

func (rt roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	endpoint := endpointGroup(r.URL.Path)
	start := time.Now()
	resp, err := rt.base.RoundTrip(r)
	rt.m.tmdbDuration.WithLabelValues(endpoint).Observe(time.Since(start).Seconds())

	status := "error"
	if err == nil {
		status = strconv.Itoa(resp.StatusCode)
	}
	rt.m.tmdbRequests.WithLabelValues(endpoint, status).Inc()
	return resp, err
}

func endpointGroup(path string) string {
	for segment := range strings.SplitSeq(strings.TrimPrefix(path, "/"), "/") {
		if segment == "" || segment == "3" {
			continue
		}
		return segment
	}
	return "other"
}

func counterFunc(name, help, cache string, fn func() float64) prometheus.Collector {
	return prometheus.NewCounterFunc(prometheus.CounterOpts{
		Namespace:   namespace,
		Name:        name,
		Help:        help,
		ConstLabels: prometheus.Labels{"cache": cache},
	}, fn)
}
