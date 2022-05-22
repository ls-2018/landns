package landns_test

import (
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestLocalCache(t *testing.T) {
	for _, tt := range CacheTests {
		tester := tt.Tester

		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			resolver := landns.NewLocalCache(CacheTestUpstream(t), landns.NewMetrics("landns"))
			defer func() {
				if err := resolver.Close(); err != nil {
					t.Fatalf("failed to close: %s", err)
				}
			}()

			tester(t, resolver)
		})
	}

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		resolver := landns.NewLocalCache(CacheTestUpstream(t), landns.NewMetrics("landns"))
		defer func() {
			if err := resolver.Close(); err != nil {
				t.Fatalf("failed to close: %s", err)
			}
		}()

		if resolver.String() != "LocalCache[0 domains 0 records]" {
			t.Errorf("unexpected string: %s", resolver)
		}

		AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 100 IN A 127.1.2.3", "example.com. 10 IN A 127.2.3.4")

		if resolver.String() != "LocalCache[1 domains 2 records]" {
			t.Errorf("unexpected string: %s", resolver)
		}
	})

	t.Run("RecursionAvailable", func(t *testing.T) {
		t.Parallel()

		CheckRecursionAvailable(t, func(rs []landns.Resolver) landns.Resolver {
			return landns.NewLocalCache(landns.ResolverSet(rs), landns.NewMetrics("landns"))
		})
	})
}

func BenchmarkLocalCache(b *testing.B) {
	upstream := landns.NewSimpleResolver([]landns.Record{
		landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 100, Address: net.ParseIP("127.1.2.3")},
	})
	if err := upstream.Validate(); err != nil {
		b.Fatalf("failed to validate upstream resolver: %s", err)
	}
	resolver := landns.NewLocalCache(upstream, landns.NewMetrics("landns"))
	defer func() {
		if err := resolver.Close(); err != nil {
			b.Fatalf("failed to close: %s", err)
		}
	}()

	req := landns.NewRequest("example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(testutil.NewDummyResponseWriter(), req)
	}

	b.StopTimer()
}
