package landns

import (
	"fmt"
	"net/http"
	"time"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is the metrics collector for the Prometheus.
type Metrics struct {
	queryCount        prometheus.Counter
	skipCount         prometheus.Counter
	resolveCounters   map[string]prometheus.Counter
	unauthCounters    map[string]prometheus.Counter
	missCounters      map[string]prometheus.Counter
	errorCounters     map[string]prometheus.Counter
	cacheHitCounters  map[string]prometheus.Counter
	cacheMissCounters map[string]prometheus.Counter
	resolveTime       prometheus.Summary
	upstreamTime      prometheus.Summary
}

func newCounter(namespace, name string, labels prometheus.Labels) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   namespace,
		Name:        fmt.Sprintf("%s_count", name),
		ConstLabels: labels,
	})
}

// NewMetrics is constructor for Metrics.
func NewMetrics(namespace string) *Metrics {
	resolves := map[string]prometheus.Counter{}
	unauthes := map[string]prometheus.Counter{}
	misses := map[string]prometheus.Counter{}
	errors := map[string]prometheus.Counter{}
	cacheHits := map[string]prometheus.Counter{}
	cacheMisses := map[string]prometheus.Counter{}

	for _, qtype := range []string{"A", "AAAA", "PTR", "SRV", "TXT"} {
		resolves[qtype] = newCounter(namespace, "resolve", prometheus.Labels{"type": qtype, "source": "local"})
		unauthes[qtype] = newCounter(namespace, "resolve", prometheus.Labels{"type": qtype, "source": "upstream"})
		misses[qtype] = newCounter(namespace, "resolve", prometheus.Labels{"type": qtype, "source": "not-found"})
		errors[qtype] = newCounter(namespace, "resolve_error", prometheus.Labels{"type": qtype})
		cacheHits[qtype] = newCounter(namespace, "cache", prometheus.Labels{"type": qtype, "cache": "hit"})
		cacheMisses[qtype] = newCounter(namespace, "cache", prometheus.Labels{"type": qtype, "cache": "miss"})
	}

	return &Metrics{
		queryCount: newCounter(namespace, "received_message", prometheus.Labels{"type": "query"}),
		skipCount:  newCounter(namespace, "received_message", prometheus.Labels{"type": "another"}),

		resolveCounters:   resolves,
		unauthCounters:    unauthes,
		missCounters:      misses,
		errorCounters:     errors,
		cacheHitCounters:  cacheHits,
		cacheMissCounters: cacheMisses,

		resolveTime: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace:  namespace,
			Name:       "resolve_duration_seconds",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),

		upstreamTime: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace:  namespace,
			Name:       "upstream_resolve_duration_seconds",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
	}
}

// HTTPHandler is make http.Handler.
func (m *Metrics) HTTPHandler() (http.Handler, error) {
	registry := prometheus.NewRegistry()

	if err := registry.Register(m); err != nil {
		return nil, Error{TypeExternalError, err, "failed to register prometheus handler"}
	}

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{}), nil
}

// Describe is register descriptions to the Prometheus.
func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.queryCount.Describe(ch)
	m.skipCount.Describe(ch)

	for _, c := range m.resolveCounters {
		c.Describe(ch)
	}
	for _, c := range m.unauthCounters {
		c.Describe(ch)
	}
	for _, c := range m.missCounters {
		c.Describe(ch)
	}
	for _, c := range m.errorCounters {
		c.Describe(ch)
	}
	for _, c := range m.cacheHitCounters {
		c.Describe(ch)
	}
	for _, c := range m.cacheMissCounters {
		c.Describe(ch)
	}

	m.resolveTime.Describe(ch)
	m.upstreamTime.Describe(ch)
}

// Collect is collect metrics to the Prometheus.
func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.queryCount.Collect(ch)
	m.skipCount.Collect(ch)

	for _, c := range m.resolveCounters {
		c.Collect(ch)
	}
	for _, c := range m.unauthCounters {
		c.Collect(ch)
	}
	for _, c := range m.missCounters {
		c.Collect(ch)
	}
	for _, c := range m.errorCounters {
		c.Collect(ch)
	}
	for _, c := range m.cacheHitCounters {
		c.Collect(ch)
	}
	for _, c := range m.cacheMissCounters {
		c.Collect(ch)
	}

	m.resolveTime.Collect(ch)
	m.upstreamTime.Collect(ch)
}

func (m *Metrics) makeTimer(skipped bool) func(*dns.Msg) {
	timer := prometheus.NewTimer(m.resolveTime)
	return func(response *dns.Msg) {
		timer.ObserveDuration()

		counters := m.resolveCounters
		if !response.Authoritative {
			counters = m.unauthCounters
		}
		if len(response.Answer) == 0 {
			counters = m.missCounters
		}

		for _, q := range response.Question {
			if counter, ok := counters[QtypeToString(q.Qtype)]; ok {
				counter.Inc()
			}
		}
	}
}

// Start is starter timer for collect resolve duration.
func (m *Metrics) Start(request *dns.Msg) func(*dns.Msg) {
	if request.Opcode != dns.OpcodeQuery {
		m.skipCount.Inc()
		return m.makeTimer(true)
	}

	m.queryCount.Inc()
	return m.makeTimer(false)
}

// Error is collector of error.
func (m *Metrics) Error(req Request, err error) {
	if counter, ok := m.errorCounters[req.QtypeString()]; ok {
		counter.Inc()
	}
}

// UpstreamTime is collector of recursion resolve.
func (m *Metrics) UpstreamTime(duration time.Duration) {
	m.upstreamTime.Observe(duration.Seconds())
}

// CacheHit is collector of cache hit rate.
func (m *Metrics) CacheHit(req Request) {
	if counter, ok := m.cacheHitCounters[req.QtypeString()]; ok {
		counter.Inc()
	}
}

// CacheMiss is collector of cache hit rate.
func (m *Metrics) CacheMiss(req Request) {
	if counter, ok := m.cacheMissCounters[req.QtypeString()]; ok {
		counter.Inc()
	}
}
