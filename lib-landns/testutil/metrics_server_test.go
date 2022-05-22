package testutil_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns/testutil"
)

func TestMetricsLabels(t *testing.T) {
	t.Parallel()

	give := `hello="world",foo="bar"`
	l := testutil.ParseMetricsLabels(give)
	s := l.String()
	if s != `{hello="world",foo="bar"}` && s != `{foo="bar",hello="world"}` {
		t.Errorf("failed to parse/string MetricsLabels\ngive:   %#v\nbut got: %#v", give, s)
	}
}

func TestMetrics(t *testing.T) {
	t.Parallel()

	ms, err := testutil.ParseMetrics(strings.Join([]string{
		`hello{target="world"} 10`,
		``,
		`# this is comment.`,
		`foo 1`,
		`bar{} 2`,
		`long_long{id="1",type="long"} 3`,
	}, "\n"))
	if err != nil {
		t.Fatalf("failed to parse metrics: %s", err)
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		tests := []testutil.MetricsEntry{
			{"hello", map[string]string{"target": "world"}, 10},
			{"foo", map[string]string{}, 1},
			{"bar", nil, 2},
			{"long_long", map[string]string{"id": "1", "type": "long"}, 3},
		}

		if len(ms) != len(tests) {
			t.Errorf("unexpected response length: expected %d but got %d", len(tests), len(ms))
		}

		for _, tt := range tests {
			ms.Assert(t, tt.Name, tt.Labels, tt.Value)
		}
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			Name   string
			Labels map[string]string
			Value  float64
			Errors []string
		}{
			{
				"hello",
				map[string]string{"target": "world"},
				9,
				[]string{"unexpected metrics value: hello{target=\"world\"}: expected 9.00 but got 10.00"},
			},
			{
				"hello",
				map[string]string{"target": "golang"},
				10,
				[]string{"expected metrics was not found: hello{target=\"golang\"}"},
			},
			{
				"not_found",
				map[string]string{},
				1,
				[]string{"expected metrics was not found: not_found{}"},
			},
		}

		for _, tt := range tests {
			tb := new(testutil.DummyTB)
			ms.Assert(tb, tt.Name, tt.Labels, tt.Value)
			tb.AssertErrors(t, tt.Errors...)
			tb.AssertFatals(t)
		}
	})
}

func TestParseMetrics_fail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Metrics string
		Error   string
	}{
		{"foobar", "failed to parse metrics: foobar"},
		{"foobar val", "strconv.ParseFloat: parsing \"val\": invalid syntax"},
	}

	for _, tt := range tests {
		if _, err := testutil.ParseMetrics(tt.Metrics); err == nil {
			t.Errorf("expected error but got nil: %s", tt.Metrics)
		} else if err.Error() != tt.Error {
			t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", tt.Error, err.Error())
		}
	}
}

func TestMetricsServer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	srv := testutil.StartMetricsServer(ctx, t, "landns")

	srv.Get(t)

	cancel()
	time.Sleep(10 * time.Millisecond)

	tb := new(testutil.DummyTB)
	srv.Get(tb)
	tb.AssertErrors(t, "failed to GET /: Get \"http://127.0.0.1:*/\": dial tcp 127.0.0.1:*: connect: connection refused")
	tb.AssertFatals(t)
}
