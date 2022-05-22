package landns_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestMetrics(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := testutil.StartMetricsServer(ctx, t, "landns")

	for i, tt := range []struct {
		Name           string
		Labels         testutil.MetricsLabels
		Authoritative  bool
		ResponseLength int
	}{
		{"landns_resolve_count", testutil.MetricsLabels{"source": "local", "type": "A"}, true, 1},
		{"landns_resolve_count", testutil.MetricsLabels{"source": "upstream", "type": "A"}, false, 1},
		{"landns_resolve_count", testutil.MetricsLabels{"source": "not-found", "type": "A"}, true, 0},
	} {
		srv.Get(t).Assert(t, tt.Name, tt.Labels, 0)
		srv.Get(t).Assert(t, "landns_received_message_count", testutil.MetricsLabels{"type": "query"}, float64(i))

		req := &dns.Msg{
			MsgHdr: dns.MsgHdr{Id: dns.Id()},
			Question: []dns.Question{
				{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			},
		}
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Authoritative = tt.Authoritative
		resp.Answer = []dns.RR{}
		for i := 0; i < tt.ResponseLength; i++ {
			rr, err := dns.NewRR(fmt.Sprintf("example.com. 42 IN A 127.0.0.%d", i))
			if err != nil {
				t.Fatalf("failed to make RR: %s", err)
			}
			resp.Answer = append(resp.Answer, rr)
		}

		srv.Metrics.Start(req)(resp)

		srv.Get(t).Assert(t, tt.Name, tt.Labels, 1)
		srv.Get(t).Assert(t, "landns_received_message_count", testutil.MetricsLabels{"type": "query"}, float64(i+1))
	}

	srv.Get(t).Assert(t, "landns_cache_count", testutil.MetricsLabels{"cache": "hit", "type": "A"}, 0)
	srv.Metrics.CacheHit(landns.NewRequest("example.com.", dns.TypeA, true))
	srv.Get(t).Assert(t, "landns_cache_count", testutil.MetricsLabels{"cache": "hit", "type": "A"}, 1)

	srv.Get(t).Assert(t, "landns_cache_count", testutil.MetricsLabels{"cache": "miss", "type": "A"}, 0)
	srv.Metrics.CacheMiss(landns.NewRequest("example.com.", dns.TypeA, true))
	srv.Get(t).Assert(t, "landns_cache_count", testutil.MetricsLabels{"cache": "miss", "type": "A"}, 1)

	srv.Get(t).Assert(t, "landns_received_message_count", testutil.MetricsLabels{"type": "another"}, 0)
	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id(), Opcode: dns.OpcodeNotify},
	}
	resp := new(dns.Msg)
	resp.SetReply(req)
	srv.Metrics.Start(req)(resp)
	srv.Get(t).Assert(t, "landns_received_message_count", testutil.MetricsLabels{"type": "another"}, 1)

	srv.Get(t).Assert(t, "landns_resolve_error_count", testutil.MetricsLabels{"type": "A"}, 0)
	srv.Metrics.Error(landns.NewRequest("example.com.", dns.TypeA, true), fmt.Errorf("test error"))
	srv.Get(t).Assert(t, "landns_resolve_error_count", testutil.MetricsLabels{"type": "A"}, 1)
}

func BenchmarkMetrics(b *testing.B) {
	metrics := landns.NewMetrics("landns")

	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{
			{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		},
	}
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	rr, err := dns.NewRR("example.com. 42 IN A 127.0.0.1")
	if err != nil {
		b.Fatalf("failed to make RR: %s", err)
	}
	resp.Answer = []dns.RR{rr}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		metrics.Start(req)(resp)
	}
}
