package benchmark

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

var (
	targets = []string{"8.8.8.8:53", "1.1.1.1:53"}
)

func NewServer(ctx context.Context, t testing.TB) *net.UDPAddr {
	t.Helper()

	metrics := landns.NewMetrics("landns")
	dyn, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %#v", err)
	}

	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5335}

	server := landns.Server{
		Metrics:         metrics,
		DynamicResolver: dyn,
		Resolvers: landns.AlternateResolver{
			landns.ResolverSet{
				landns.NewSimpleResolver([]landns.Record{}),
				dyn,
			},
			landns.NewLocalCache(landns.NewForwardResolver([]*net.UDPAddr{
				{IP: net.ParseIP("8.8.8.8"), Port: 53},
			}, 100*time.Millisecond, metrics), metrics),
		},
	}

	go func() {
		err := server.ListenAndServe(ctx, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5335}, addr, "udp")
		if err != nil {
			t.Fatalf("failed to start server: %s", err)
		}
	}()

	time.Sleep(100 * time.Millisecond) // wait for start server

	return addr
}

func BenchmarkDNS(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := NewServer(ctx, b)

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{
			{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		},
	}

	for _, target := range append(targets, server.String()) {
		b.Run(target, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				dns.Exchange(msg, target)
			}
		})
	}
}
