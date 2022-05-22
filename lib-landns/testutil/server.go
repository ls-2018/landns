package testutil

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/macrat/landns/client/go-client"
	"github.com/macrat/landns/lib-landns"
)

// StartServer is make landns.Server and start it.
func StartServer(ctx context.Context, t SimpleTB, debugMode bool) (client.Client, *net.UDPAddr) {
	metrics := landns.NewMetrics("landns")
	dyn, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %s", err)
	}

	s := &landns.Server{
		Metrics:         metrics,
		DynamicResolver: dyn,
		Resolvers:       dyn,
		DebugMode:       debugMode,
	}

	apiAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: FindEmptyPort()}
	dnsAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3553}
	go func() {
		if err := s.ListenAndServe(ctx, apiAddr, dnsAddr, "udp"); err != nil {
			t.Fatalf("failed to start server: %s", err)
		}
	}()

	u, err := url.Parse(fmt.Sprintf("http://%s/api/v1/", apiAddr))
	if err != nil {
		t.Fatalf("failed to parse URL: %s", err)
	}

	time.Sleep(10 * time.Millisecond) // wait for start server

	return client.New(u), dnsAddr
}
