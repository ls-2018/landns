package testutil

import (
	"context"
	"net"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

// DNSServer is tester for DNS blackbox test.
type DNSServer struct {
	Addr *net.UDPAddr
}

// StartDNSServer is make dns.Server and start it.
func StartDNSServer(ctx context.Context, t SimpleTB, resolver landns.Resolver) DNSServer {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: FindEmptyPort()}

	server := dns.Server{
		Addr:      addr.String(),
		Net:       "udp",
		ReusePort: true,
		Handler:   landns.NewHandler(resolver, landns.NewMetrics("landns")),
	}

	go func() {
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			t.Fatalf("failed to serve dummy DNS: %s", err)
		}
	}()

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(); err != nil {
			t.Fatalf("failed to stop dummy DNS: %s", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // Wait for start DNS server

	return DNSServer{addr}
}

// Assert is assertion tester for dns message exchange.
func (d DNSServer) Assert(t SimpleTB, q dns.Question, expect ...string) {
	t.Helper()

	msg := &dns.Msg{
		MsgHdr:   dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{q},
	}

	in, err := dns.Exchange(msg, d.Addr.String())
	if err != nil {
		t.Errorf("%s: failed to resolve: %s", d.Addr, err)
		return
	}

	ok := len(in.Answer) == len(expect)

	if ok {
		for i := range expect {
			if in.Answer[i].String() != expect[i] {
				ok = false
				break
			}
		}
	}

	if !ok {
		msg := "%s: unexpected answer:\nexpected:\n"
		for _, x := range expect {
			msg += "\t" + x + "\n"
		}
		msg += "but got:\n"
		for _, x := range in.Answer {
			msg += "\t" + x.String() + "\n"
		}
		t.Errorf(msg, d.Addr)
	}
}
