package landns_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestForwardResolver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCases := map[landns.Request][]landns.Record{
		landns.NewRequest("example.com.", dns.TypeA, false): {},
		landns.NewRequest("example.com.", dns.TypeA, true): {
			landns.AddressRecord{Name: "example.com.", TTL: 123, Address: net.ParseIP("127.0.0.1")},
			landns.AddressRecord{Name: "example.com.", TTL: 456, Address: net.ParseIP("127.0.0.2")},
		},
		landns.NewRequest("example.com.", dns.TypeAAAA, true): {
			landns.AddressRecord{Name: "example.com.", TTL: 789, Address: net.ParseIP("1::4:2")},
		},
		landns.NewRequest("file.example.com.", dns.TypeCNAME, true): {
			landns.CnameRecord{Name: "file.example.com.", TTL: 123, Target: "example.com."},
		},
		landns.NewRequest("127.0.0.1.in-addr.arpa.", dns.TypePTR, true): {
			landns.PtrRecord{Name: "127.0.0.1.in-addr.arpa.", TTL: 234, Domain: "example.com."},
		},
		landns.NewRequest("_web._tcp.example.com.", dns.TypeSRV, true): {
			landns.SrvRecord{
				Name:     "_web._tcp.example.com.",
				TTL:      234,
				Priority: 1,
				Weight:   2,
				Port:     3,
				Target:   "file.example.com.",
			},
		},
		landns.NewRequest("example.com.", dns.TypeTXT, true): {
			landns.TxtRecord{Name: "example.com.", TTL: 234, Text: "hello world"},
		},
	}
	records := []landns.Record{}
	for _, rs := range testCases {
		records = append(records, rs...)
	}
	srv := testutil.StartDNSServer(ctx, t, landns.NewSimpleResolver(records))

	resolver := landns.NewForwardResolver([]*net.UDPAddr{{IP: net.ParseIP("127.0.0.1"), Port: 5321}, srv.Addr}, 1*time.Second, landns.NewMetrics("landns"))
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	for request, records := range testCases {
		rs := []string{}
		for _, r := range records {
			rs = append(rs, r.String())
		}
		AssertResolve(t, resolver, request, !request.RecursionDesired, rs...)
	}

	if resolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recursion available: %v", resolver.RecursionAvailable())
	}
}

func TestForwardResolver_Parallel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := testutil.StartDNSServer(ctx, t, landns.NewSimpleResolver([]landns.Record{}))

	resolver := landns.NewForwardResolver([]*net.UDPAddr{srv.Addr}, 1*time.Second, landns.NewMetrics("landns"))
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	ParallelResolveTest(t, resolver)
}
