package landns_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/logger"
	"github.com/macrat/landns/lib-landns/logger/logtest"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := landns.NewSimpleResolver([]landns.Record{
		landns.AddressRecord{Name: "example.com.", TTL: 123, Address: net.ParseIP("127.0.0.1")},
	})
	if err := resolver.Validate(); err != nil {
		t.Errorf("failed to make resolver: %s", err)
	}

	srv := testutil.StartDNSServer(ctx, t, resolver)

	lt := logtest.Start()
	defer lt.Close()

	srv.Assert(t, dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}, "example.com.\t123\tIN\tA\t127.0.0.1")
	time.Sleep(10 * time.Millisecond)

	if err := lt.TestAll([]logtest.Entry{}); err != nil {
		t.Error(err)
	}

	srv.Assert(t, dns.Question{Name: "notfound.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET})
	time.Sleep(10 * time.Millisecond)

	if err := lt.Test([]logtest.Entry{{Level: logger.InfoLevel, Message: "not found", Fields: logger.Fields{"proto": "dns", "name": "notfound.example.com.", "type": "A"}}}); err != nil {
		t.Error(err)
	}

	srv.Assert(t, dns.Question{Name: "notfound.example.com.", Qtype: dns.TypeAAAA, Qclass: dns.ClassINET})
	time.Sleep(10 * time.Millisecond)

	if err := lt.Test([]logtest.Entry{{Level: logger.InfoLevel, Message: "not found", Fields: logger.Fields{"proto": "dns", "name": "notfound.example.com.", "type": "AAAA"}}}); err != nil {
		t.Error(err)
	}
}

func TestHandler_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := &testutil.DummyResolver{Error: false, Recursion: false}
	srv := testutil.StartDNSServer(ctx, t, resolver)

	lt := logtest.Start()
	defer lt.Close()

	srv.Assert(t, dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET})

	if err := lt.Test([]logtest.Entry{{Level: logger.InfoLevel, Message: "not found", Fields: logger.Fields{"proto": "dns", "name": "example.com.", "type": "A"}}}); err != nil {
		t.Error(err)
	}

	resolver.Error = true

	srv.Assert(t, dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET})

	if err := lt.Test([]logtest.Entry{{Level: logger.WarnLevel, Message: "failed to resolve", Fields: logger.Fields{"proto": "dns", "reason": "test error", "name": "example.com.", "type": "A"}}}); err != nil {
		t.Error(err)
	}
}
