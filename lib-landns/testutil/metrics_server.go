package testutil

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/macrat/landns/lib-landns"
)

// MetricsLabels is labels for MetricsEntry.
type MetricsLabels map[string]string

// ParseMetricsLabels is parser to MetricsLabels.
func ParseMetricsLabels(str string) MetricsLabels {
	ms := regexp.MustCompile(`([a-z_]+)="([^"]*)"`).FindAllStringSubmatch(str, -1)

	ls := make(MetricsLabels)
	for _, m := range ms {
		ls[m[1]] = m[2]
	}

	return ls
}

// String is human readable string getter.
func (l MetricsLabels) String() string {
	xs := make([]string, 0, len(l))
	for k, v := range l {
		xs = append(xs, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return "{" + strings.Join(xs, ",") + "}"
}

// MetricsEntry is one entry of Metrics.
type MetricsEntry struct {
	Name   string
	Labels MetricsLabels
	Value  float64
}

// Metrics is response data from MetricsServer.
type Metrics []MetricsEntry

// ParseMetrics is parser for Metrics.
func ParseMetrics(str string) (Metrics, error) {
	re := regexp.MustCompile(`^([a-z_]+)((?:\{[^}]*\})?)\s+(.*)$`)
	var ms Metrics

	for _, line := range strings.Split(str, "\n") {
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		m := re.FindStringSubmatch(line)
		if len(m) != 4 {
			return nil, fmt.Errorf("failed to parse metrics: %s", line)
		}

		v, err := strconv.ParseFloat(m[3], 64)
		if err != nil {
			return nil, err
		}
		ms = append(ms, MetricsEntry{
			Name:   m[1],
			Labels: ParseMetricsLabels(m[2]),
			Value:  v,
		})
	}

	return ms, nil
}

// Assert is assertion test for Metrics data.
func (ms Metrics) Assert(t SimpleTB, name string, labels MetricsLabels, value float64) {
	t.Helper()

	for _, m := range ms {
		if m.Name != name || len(m.Labels) != len(labels) {
			continue
		}

		ok := true
		for k := range m.Labels {
			if m.Labels[k] != labels[k] {
				ok = false
				break
			}
		}
		if ok {
			if m.Value != value {
				t.Errorf("unexpected metrics value: %s%s: expected %.2f but got %.2f", name, labels, value, m.Value)
			}
			return
		}
	}
	t.Errorf("expected metrics was not found: %s%s", name, labels)
}

// MetricsServer is tester for Metrics blackbox test.
type MetricsServer struct {
	Metrics    *landns.Metrics
	HTTPServer HTTPServer
}

// StartMetricsServer is make metrics server and start it.
func StartMetricsServer(ctx context.Context, t SimpleTB, namespace string) MetricsServer {
	t.Helper()

	metrics := landns.NewMetrics("landns")
	handler, err := metrics.HTTPHandler()
	if err != nil {
		t.Fatalf("failed to serve dummy metrics server: %s", err)
	}

	return MetricsServer{metrics, StartHTTPServer(ctx, t, handler)}
}

// Get is get Metrics from MetricsServer.
func (s MetricsServer) Get(t SimpleTB) Metrics {
	ms, err := ParseMetrics(s.HTTPServer.Do(t, "GET", "/", "").Body)
	if err != nil {
		t.Fatalf("failed to get metrics: %s", err)
	}
	return ms
}
