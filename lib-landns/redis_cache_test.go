package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

var (
	redisAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6379}
)

func prepareRedisDB(t testing.TB) {
	t.Helper()

	conn, err := redis.Dial(redisAddr.Network(), redisAddr.String())
	if err != nil {
		t.Skip("redis server was not found")
	}
	defer conn.Close()

	if err := conn.Send("FLUSHALL"); err != nil {
		t.Fatalf("failed to flush database: %s", err)
	}
	if err := conn.Flush(); err != nil {
		t.Fatalf("failed to flush database: %s", err)
	}
}

func TestRedisCache(t *testing.T) {
	for _, tt := range CacheTests {
		tester := tt.Tester

		t.Run(tt.Name, func(t *testing.T) {
			prepareRedisDB(t)

			resolver, err := landns.NewRedisCache(redisAddr, 0, "", CacheTestUpstream(t), landns.NewMetrics("landns"))
			if err != nil {
				t.Fatalf("failed to connect redis server: %s", err)
			}
			defer func() {
				if err := resolver.Close(); err != nil {
					t.Fatalf("failed to close: %s", err)
				}
			}()

			if resolver.String() != fmt.Sprintf("RedisCache[%s, SimpleResolver[3 domains 2 types 6 records]]", redisAddr) {
				t.Errorf("unexpected string: %s", resolver)
			}

			tester(t, resolver)
		})
	}

	t.Run("RecursionAvailable", func(t *testing.T) {
		prepareRedisDB(t)

		CheckRecursionAvailable(t, func(rs []landns.Resolver) landns.Resolver {
			resolver, err := landns.NewRedisCache(redisAddr, 0, "", landns.ResolverSet(rs), landns.NewMetrics("landns"))
			if err != nil {
				t.Fatalf("failed to connect redis server: %s", err)
			}
			return resolver
		})
	})

	t.Run("failedToConnect", func(t *testing.T) {
		expect := "failed to connect to Redis server: dial tcp :0: connect: connection refused"

		_, err := landns.NewRedisCache(&net.TCPAddr{}, 0, "", testutil.DummyResolver{}, landns.NewMetrics("landns"))
		if err == nil {
			t.Errorf("expected error but got nil")
		} else if err.Error() != expect {
			t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", expect, err.Error())
		}
	})
}

func BenchmarkRedisCache(b *testing.B) {
	prepareRedisDB(b)

	upstream := landns.NewSimpleResolver([]landns.Record{
		landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 100, Address: net.ParseIP("127.1.2.3")},
	})
	if err := upstream.Validate(); err != nil {
		b.Fatalf("failed to validate upstream resolver: %s", err)
	}
	resolver, err := landns.NewRedisCache(redisAddr, 0, "", upstream, landns.NewMetrics("landns"))
	if err != nil {
		b.Fatalf("failed to connect redis server: %s", err)
	}
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
