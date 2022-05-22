package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestSimpleResolver(t *testing.T) {
	t.Parallel()

	resolver := landns.NewSimpleResolver(
		[]landns.Record{
			landns.AddressRecord{Name: landns.Domain("example.com."), Address: net.ParseIP("127.1.2.3")},
			landns.AddressRecord{Name: landns.Domain("example.com."), Address: net.ParseIP("127.2.3.4")},
			landns.AddressRecord{Name: landns.Domain("blanktar.jp."), Address: net.ParseIP("127.2.2.2")},
			landns.AddressRecord{Name: landns.Domain("blanktar.jp."), Address: net.ParseIP("4::2")},
			landns.TxtRecord{Name: landns.Domain("example.com."), Text: "hello"},
			landns.TxtRecord{Name: landns.Domain("blanktar.jp."), Text: "foo"},
			landns.TxtRecord{Name: landns.Domain("blanktar.jp."), Text: "bar"},
			landns.PtrRecord{Name: landns.Domain("3.2.1.127.in-addr.arpa."), Domain: landns.Domain("target.local.")},
			landns.PtrRecord{Name: landns.Domain("8.7.6.5.4.3.2.1.f.e.d.c.b.a.0.9.8.7.6.5.4.3.2.1.ip6.arpa."), Domain: landns.Domain("target.local.")},
			landns.CnameRecord{Name: landns.Domain("example.com."), Target: landns.Domain("target.local.")},
			landns.SrvRecord{Name: landns.Domain("_http._tcp.example.com."), Port: 10, Target: landns.Domain("target.local.")},
			landns.MxRecord{Name: landns.Domain("example.com."), Preference: 10, Target: landns.Domain("mail.example.com.")},
			landns.NsRecord{Name: landns.Domain("example.com."), Target: landns.Domain("ns1.example.com.")},
		},
	)
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	if err := resolver.Validate(); err != nil {
		t.Fatalf("failed to validate resolver: %s", err)
	}

	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 0 IN A 127.1.2.3", "example.com. 0 IN A 127.2.3.4")
	AssertResolve(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeA, false), true, "blanktar.jp. 0 IN A 127.2.2.2")
	AssertResolve(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeAAAA, false), true, "blanktar.jp. 0 IN AAAA 4::2")
	AssertResolve(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeA, false), true)
	AssertResolve(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeAAAA, false), true)

	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true, `example.com. 0 IN TXT "hello"`)
	AssertResolve(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeTXT, false), true, `blanktar.jp. 0 IN TXT "foo"`, `blanktar.jp. 0 IN TXT "bar"`)
	AssertResolve(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeTXT, false), true)

	AssertResolve(t, resolver, landns.NewRequest("3.2.1.127.in-addr.arpa.", dns.TypePTR, false), true, "3.2.1.127.in-addr.arpa. 0 IN PTR target.local.")
	AssertResolve(t, resolver, landns.NewRequest("8.7.6.5.4.3.2.1.f.e.d.c.b.a.0.9.8.7.6.5.4.3.2.1.ip6.arpa.", dns.TypePTR, false), true, "8.7.6.5.4.3.2.1.f.e.d.c.b.a.0.9.8.7.6.5.4.3.2.1.ip6.arpa. 0 IN PTR target.local.")
	AssertResolve(t, resolver, landns.NewRequest("4.2.1.127.in-addr.arpa.", dns.TypePTR, false), true)

	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeCNAME, false), true, "example.com. 0 IN CNAME target.local.")
	AssertResolve(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeCNAME, false), true)

	AssertResolve(t, resolver, landns.NewRequest("_http._tcp.example.com.", dns.TypeSRV, false), true, "_http._tcp.example.com. 0 IN SRV 0 0 10 target.local.")
	AssertResolve(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeSRV, false), true)

	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeMX, false), true, "example.com. 0 IN MX 10 mail.example.com.")

	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeNS, false), true, "example.com. IN NS ns1.example.com.")
}

func TestSimpleResolver_Parallel(t *testing.T) {
	t.Parallel()

	resolver := landns.NewSimpleResolver([]landns.Record{})

	ParallelResolveTest(t, resolver)
}

func BenchmarkSimpleResolver(b *testing.B) {
	records := []landns.Record{}

	for i := 0; i < 100; i++ {
		host := landns.Domain(fmt.Sprintf("host%d.example.com.", i))
		records = append(records, landns.AddressRecord{Name: host, Address: net.ParseIP("127.1.2.3")})
		records = append(records, landns.AddressRecord{Name: host, Address: net.ParseIP("127.2.3.4")})
	}

	resolver := landns.NewSimpleResolver(records)
	defer func() {
		if err := resolver.Close(); err != nil {
			b.Fatalf("failed to close: %s", err)
		}
	}()

	if err := resolver.Validate(); err != nil {
		b.Fatalf("failed to validate resolver: %s", err)
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(testutil.NewDummyResponseWriter(), req)
	}

	b.StopTimer()
}

func TestNewSimpleResolverFromConfig(t *testing.T) {
	t.Parallel()

	config := []byte(`ttl: 128

address:
  example.com: [127.1.2.3]
  server.example.com.:
    - 192.168.1.2
    - 192.168.1.3
    - 1:2::3

cname:
  file.example.com: [server.example.com.]

text:
  example.com:
    - hello world
    - foo

service:
  example.com:
    - service: ftp
      proto: tcp
      priority: 1
      weight: 2
      port: 21
      target: file.example.com
    - service: http
      port: 80
      target: server.example.com
`)

	resolver, err := landns.NewSimpleResolverFromConfig(config)
	if err != nil {
		t.Fatalf("failed to parse config: %s", err.Error())
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	if err := resolver.Validate(); err != nil {
		t.Fatalf("invalid resolver state: %s", err)
	}

	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 128 IN A 127.1.2.3")
	AssertResolve(t, resolver, landns.NewRequest("server.example.com.", dns.TypeA, false), true, "server.example.com. 128 IN A 192.168.1.2", "server.example.com. 128 IN A 192.168.1.3")

	AssertResolve(t, resolver, landns.NewRequest("server.example.com.", dns.TypeAAAA, false), true, "server.example.com. 128 IN AAAA 1:2::3")

	AssertResolve(t, resolver, landns.NewRequest("3.2.1.127.in-addr.arpa.", dns.TypePTR, false), true, "3.2.1.127.in-addr.arpa. 128 IN PTR example.com.")
	AssertResolve(t, resolver, landns.NewRequest("2.1.168.192.in-addr.arpa.", dns.TypePTR, false), true, "2.1.168.192.in-addr.arpa. 128 IN PTR server.example.com.")
	AssertResolve(t, resolver, landns.NewRequest("3.1.168.192.in-addr.arpa.", dns.TypePTR, false), true, "3.1.168.192.in-addr.arpa. 128 IN PTR server.example.com.")

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("3.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.0.1.0.0.0.ip6.arpa.", dns.TypePTR, false),
		true,
		"3.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.0.1.0.0.0.ip6.arpa. 128 IN PTR server.example.com.",
	)

	AssertResolve(t, resolver, landns.NewRequest("file.example.com.", dns.TypeCNAME, false), true, "file.example.com. 128 IN CNAME server.example.com.")
	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true, `example.com. 128 IN TXT "hello world"`, `example.com. 128 IN TXT "foo"`)

	AssertResolve(t, resolver, landns.NewRequest("_ftp._tcp.example.com.", dns.TypeSRV, false), true, "_ftp._tcp.example.com. 128 IN SRV 1 2 21 file.example.com.")
	AssertResolve(t, resolver, landns.NewRequest("_http._tcp.example.com.", dns.TypeSRV, false), true, "_http._tcp.example.com. 128 IN SRV 0 0 80 server.example.com.")
}

func TestNewSimpleResolverFromConfig_WithoutTTL(t *testing.T) {
	t.Parallel()

	config := []byte(`address: {example.com: [127.1.2.3]}`)

	resolver, err := landns.NewSimpleResolverFromConfig(config)
	if err != nil {
		t.Fatalf("failed to parse config: %s", err.Error())
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 3600 IN A 127.1.2.3")
}
