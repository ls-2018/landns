package landns_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/logger/logtest"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, d := testutil.StartServer(ctx, t, false)

	if rs, err := landns.NewDynamicRecordSet("example.com. 300 IN A 127.0.1.2"); err != nil {
		t.Errorf("failed to parse record set: %s", err)
	} else if err := c.Set(rs); err != nil {
		t.Errorf("failed to set dynamic record: %s", err)
	}

	if rs, err := c.Get(); err != nil {
		t.Errorf("failed to get records: %s", err)
	} else if len(rs) != 2 {
		t.Errorf("unexpected records:\n%s", rs)
	}

	AssertExchange(t, d, []dns.Question{
		{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}, "example.com.\t300\tIN\tA\t127.0.1.2")

	if u, err := c.Endpoint.Parse("/"); err != nil {
		t.Errorf("failed to make root url: %s", err)
	} else if resp, err := http.Get(u.String()); err != nil {
		t.Errorf("failed to get index page: %s", err)
	} else {
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		expect := `<h1>Landns</h1><a href="/metrics">metrics</a> <a href="/api/v1">records</a>` + "\n"
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("failed to read index page: %s", err)
		} else if string(body) != expect {
			t.Errorf("unexpected index page:\nexpected:\n%s\nbut got:\n%s\n", expect, string(body))
		}
	}

	if u, err := c.Endpoint.Parse("/metrics"); err != nil {
		t.Errorf("failed to make metrics url: %s", err)
	} else if resp, err := http.Get(u.String()); err != nil {
		t.Errorf("failed to get metrics: %s", err)
	} else if resp.StatusCode != 200 {
		t.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestServer_StartStop(t *testing.T) {
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		testutil.StartServer(ctx, t, false)

		cancel()

		time.Sleep(10 * time.Millisecond) // wait for stop server
	}
}

func TestServer_DebugMode(t *testing.T) {
	tests := []struct {
		DebugMode bool
		RootPage  string
	}{
		{false, `<h1>Landns</h1><a href="/metrics">metrics</a> <a href="/api/v1">records</a>` + "\n"},
		{true, `<h1>Landns</h1><a href="/metrics">metrics</a> <a href="/debug/pprof/">pprof</a> <a href="/api/v1">records</a>` + "\n"},
	}

	for _, tt := range tests {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c, _ := testutil.StartServer(ctx, t, tt.DebugMode)

		if u, err := c.Endpoint.Parse("/"); err != nil {
			t.Errorf("failed to make root url: %s", err)
		} else if resp, err := http.Get(u.String()); err != nil {
			t.Errorf("failed to get index page: %s", err)
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("failed to read index page: %s", err)
			} else if string(body) != tt.RootPage {
				t.Errorf("unexpected index page:\nexpected:\n%s\nbut got:\n%s\n", tt.RootPage, string(body))
			}
		}

		if u, err := c.Endpoint.Parse("/debug/pprof/"); err != nil {
			t.Errorf("failed to make pprof url: %s", err)
		} else if resp, err := http.Get(u.String()); err != nil {
			t.Errorf("failed to get pprof page: %s", err)
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("failed to read pprof page: %s", err)
			} else {
				if tt.DebugMode && string(body) == tt.RootPage {
					t.Errorf("seems to not served pprof page in debug mode")
				}
				if !tt.DebugMode && string(body) != tt.RootPage {
					t.Errorf("seems to served pprof page in production mode")
				}
			}
		}
	}
}

func TestServer_logging(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lt := logtest.Start()
	defer lt.Close()

	c, _ := testutil.StartServer(ctx, t, false)

	if ln := len(*lt.Logger); ln != 0 {
		t.Fatalf("unexpected log length: %d\n%s", ln, lt.Logger)
	}

	if _, err := c.Get(); err != nil {
		t.Errorf("failed to get records: %s", err)
	}

	if ln := len(*lt.Logger); ln != 1 {
		t.Errorf("unexpected log length: %d\n%s", ln, lt.Logger)
	}
}
