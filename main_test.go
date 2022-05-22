package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/logger"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func MakeDummyFile(content string) (closer func(), path string, err error) {
	tmp, err := ioutil.TempFile("", "landns_test_")
	if err != nil {
		return nil, "", err
	}

	fmt.Fprint(tmp, content)

	return func() {
		os.Remove(tmp.Name())
	}, tmp.Name(), nil
}

func TestLoadStaticResolvers(t *testing.T) {
	closer, pathA, err := MakeDummyFile(`ttl: 10
address:
  example.com.: [127.0.1.2]`)
	if err != nil {
		t.Fatalf("failed to make dummy file: %s", err)
	}
	defer closer()

	closer, pathB, err := MakeDummyFile(`ttl: 20
address:
  example.com.: [127.1.2.3]`)
	if err != nil {
		t.Fatalf("failed to make dummy file: %s", err)
	}
	defer closer()

	resolver, err := loadStatisResolvers([]string{pathA, pathB})
	if err != nil {
		t.Fatalf("failed to load configs: %s", err)
	}

	if len(resolver) != 2 {
		t.Fatalf("unexpected length of resolver set: expected 2 but got %d", len(resolver))
	}

	records := []landns.Record{}
	writer := landns.NewResponseCallback(func(r landns.Record) error {
		records = append(records, r)
		return nil
	})
	err = resolver.Resolve(writer, landns.NewRequest("example.com.", dns.TypeA, false))
	if err != nil {
		t.Fatalf("failed to resolve: %s", err)
	}

	if len(records) != 2 {
		t.Fatalf("unexpected response length: expected 2 but got %d", len(records))
	}

	if records[0].String() != "example.com. 10 IN A 127.0.1.2" {
		t.Errorf("unexpected response: %s", records[0])
	}

	if records[1].String() != "example.com. 20 IN A 127.1.2.3" {
		t.Errorf("unexpected response: %s", records[1])
	}
}

func startServer(t *testing.T, args []string) (*service, func()) {
	service, err := makeServer(args)
	if err != nil {
		t.Fatalf("failed to make server: %s", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := service.Start(ctx); err != nil {
			t.Errorf("failed to start server: %s", err)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	return service, func() {
		if err := service.Stop(); err != nil {
			t.Fatalf("failed to close resolver: %s", err)
		}
		cancel()
	}
}

func TestMakeServer(t *testing.T) {
	t.Run("simple/make", func(t *testing.T) {
		service, err := makeServer([]string{})
		if err != nil {
			t.Fatalf("failed to make server: %s", err)
		}
		if err := service.Stop(); err != nil {
			t.Fatalf("failed to close resolver: %s", err)
		}
	})

	t.Run("simple/run", func(t *testing.T) {
		_, cancel := startServer(t, []string{"-l", fmt.Sprintf("127.0.0.1:%d", testutil.FindEmptyPort()), "-L", ":1053"})
		defer cancel()
	})
	t.Run("static", func(t *testing.T) {
		closer, path, err := MakeDummyFile("ttl: 10\naddress:\n  example.com.: [127.0.1.2]\n")
		if err != nil {
			t.Fatalf("failed to make dummy file: %s", err)
		}
		defer closer()

		_, cancel := startServer(t, []string{"-l", fmt.Sprintf("127.0.0.1:%d", testutil.FindEmptyPort()), "-L", "127.0.0.1:1053", "-c", path})
		defer cancel()

		msg := &dns.Msg{
			MsgHdr: dns.MsgHdr{Id: dns.Id()},
			Question: []dns.Question{
				{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			},
		}
		in, err := dns.Exchange(msg, "127.0.0.1:1053")
		if err != nil {
			t.Fatalf("failed to resolve google.com.: %s", err)
		}
		if in == nil {
			t.Fatalf("failed to resolve google.com.: returns nil")
		}

		expected := "example.com.\t10\tIN\tA\t127.0.1.2"
		if len(in.Answer) != 1 || in.Answer[0].String() != expected {
			t.Errorf("unexpected response:\nexpected: [%s]\nbut got:  %s", expected, in.Answer)
		}
	})
	t.Run("upstream", func(t *testing.T) {
		_, cancel := startServer(t, []string{"-l", fmt.Sprintf("127.0.0.1:%d", testutil.FindEmptyPort()), "-L", "127.0.0.1:1053", "-u", "8.8.8.8:53", "-u", "8.8.4.4:53", "-u", "1.1.1.1:53"})
		defer cancel()

		msg := &dns.Msg{
			MsgHdr: dns.MsgHdr{Id: dns.Id()},
			Question: []dns.Question{
				{Name: "google.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			},
		}
		if _, err := dns.Exchange(msg, "127.0.0.1:1053"); err != nil {
			t.Errorf("failed to resolve google.com.: %s", err)
		}
	})
}

func TestLogging(t *testing.T) {
	tests := []struct {
		Args   []string
		Expect logger.Level
	}{
		{[]string{}, logger.WarnLevel},
		{[]string{"-v"}, logger.InfoLevel},
		{[]string{"--verbose"}, logger.InfoLevel},
	}

	for _, tt := range tests {
		_, err := makeServer(tt.Args)
		if err != nil {
			t.Fatalf("failed to make server: %s", err)
		}

		level := logger.Level(logger.GetLogger().(*logger.BasicLogger).Logger.GetLevel())
		if level != tt.Expect {
			t.Errorf("unexpected logging level: expected %s but got %s", tt.Expect, level)
		}
	}
}
