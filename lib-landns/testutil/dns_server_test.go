package testutil_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestDNSServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	resolver := landns.NewSimpleResolver([]landns.Record{
		landns.AddressRecord{Name: "example.com.", TTL: 123, Address: net.ParseIP("127.0.1.2")},
	})
	srv := testutil.StartDNSServer(ctx, t, resolver)

	tests := []struct {
		Question dns.Question
		Answer   []string
		Errors   []string
	}{
		{
			dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			[]string{"example.com.\t123\tIN\tA\t127.0.1.2"},
			[]string{},
		},
		{
			dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			[]string{"example.com.\t999\tIN\tA\t127.0.1.2"},
			[]string{strings.Join([]string{
				"127.0.0.1:*: unexpected answer:",
				"expected:",
				"\texample.com.\t999\tIN\tA\t127.0.1.2",
				"but got:",
				"\texample.com.\t123\tIN\tA\t127.0.1.2",
				"",
			}, "\n")},
		},
		{
			dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			[]string{"example.com.\t123\tIN\tA\t127.0.1.2", "example.com.\t456\tIN\tA\t127.3.4.5"},
			[]string{strings.Join([]string{
				"127.0.0.1:*: unexpected answer:",
				"expected:",
				"\texample.com.\t123\tIN\tA\t127.0.1.2",
				"\texample.com.\t456\tIN\tA\t127.3.4.5",
				"but got:",
				"\texample.com.\t123\tIN\tA\t127.0.1.2",
				"",
			}, "\n")},
		},
		{
			dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			[]string{},
			[]string{strings.Join([]string{
				"127.0.0.1:*: unexpected answer:",
				"expected:",
				"but got:",
				"\texample.com.\t123\tIN\tA\t127.0.1.2",
				"",
			}, "\n")},
		},
	}

	for _, tt := range tests {
		tb := new(testutil.DummyTB)
		srv.Assert(tb, tt.Question, tt.Answer...)
		tb.AssertErrors(t, tt.Errors...)
		tb.AssertFatals(t)
	}

	cancel()
	time.Sleep(10 * time.Millisecond)

	tb := new(testutil.DummyTB)
	srv.Assert(tb, dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}, "example.com.\t123\tIN\tA\t127.0.1.2")
	tb.AssertErrors(t, "127.0.0.1:*: failed to resolve: read udp 127.0.0.1:*->127.0.0.1:*: read: connection refused")
	tb.AssertFatals(t)
}
