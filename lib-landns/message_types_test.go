package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestRequest(t *testing.T) {
	t.Parallel()

	x := landns.NewRequest("example.com.", dns.TypeA, true)
	if x.QtypeString() != "A" {
		t.Errorf("unexpected qtype string: %s", x.QtypeString())
	}
	if x.String() != ";example.com. IN A" {
		t.Errorf(`unexpected string: "%s"`, x.String())
	}
	if x.Question.String() != ";example.com.\tIN\t A" {
		t.Errorf(`unexpected string of request as dns.Question: "%s"`, x.Question.String())
	}

	y := landns.NewRequest("example.com.", dns.TypeCNAME, true)
	if y.QtypeString() != "CNAME" {
		t.Errorf("unexpected qtype string: %s", y.QtypeString())
	}
	if y.String() != ";example.com. IN CNAME" {
		t.Errorf(`unexpected string: "%s"`, y.String())
	}
	if y.Question.String() != ";example.com.\tIN\t CNAME" {
		t.Errorf(`unexpected string of request as dns.Question: "%s"`, y.Question.String())
	}
}

func TestResponseCallback(t *testing.T) {
	t.Parallel()

	rc := landns.NewResponseCallback(func(r landns.Record) error {
		return fmt.Errorf("test error")
	})

	if rc.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: %v", rc.IsAuthoritative())
	}
	rc.SetNoAuthoritative()
	if rc.IsAuthoritative() != false {
		t.Errorf("unexpected authoritative: %v", rc.IsAuthoritative())
	}

	if err := rc.Add(landns.AddressRecord{}); err == nil {
		t.Errorf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Errorf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}

	log := make([]landns.Record, 0, 5)
	rc = landns.NewResponseCallback(func(r landns.Record) error {
		log = append(log, r)
		return nil
	})
	for i := 0; i < 5; i++ {
		if len(log) != i {
			t.Errorf("unexpected log length: expected %d but got %d", i, len(log))
		}

		text := fmt.Sprintf("test%d", i)
		if err := rc.Add(landns.TxtRecord{Text: text}); err != nil {
			t.Errorf("failed to add record: %s", err)
		}

		if len(log) != i+1 {
			t.Errorf("unexpected log length: expected %d but got %d", i, len(log))
		} else if tr, ok := log[i].(landns.TxtRecord); !ok {
			t.Errorf("unexpected record type: %#v", log[i])
		} else if tr.Text != text {
			t.Errorf(`unexpected text: expected "%s" but got "%s"`, text, tr.Text)
		}
	}
}

func TestResponseWriterHook(t *testing.T) {
	t.Parallel()

	upstreamLog := make([]landns.Record, 0, 5)
	upstream := landns.NewResponseCallback(func(r landns.Record) error {
		upstreamLog = append(upstreamLog, r)
		return nil
	})

	hookLog := make([]landns.Record, 0, 5)
	hook := landns.ResponseWriterHook{
		Writer: upstream,
		OnAdd: func(r landns.Record) error {
			hookLog = append(hookLog, r)
			return nil
		},
	}

	if upstream.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative of upstream: %v", upstream.IsAuthoritative())
	}
	if hook.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative of hook: %v", hook.IsAuthoritative())
	}
	hook.SetNoAuthoritative()
	if upstream.IsAuthoritative() != false {
		t.Errorf("unexpected authoritative of upstream: %v", upstream.IsAuthoritative())
	}
	if hook.IsAuthoritative() != false {
		t.Errorf("unexpected authoritative of hook: %v", hook.IsAuthoritative())
	}

	for i := 0; i < 5; i++ {
		if len(upstreamLog) != i {
			t.Errorf("unexpected upstream log length: expected %d but got %d", i, len(upstreamLog))
		}
		if len(hookLog) != i {
			t.Errorf("unexpected hook log length: expected %d but got %d", i, len(hookLog))
		}

		text := fmt.Sprintf("test%d", i)
		if err := hook.Add(landns.TxtRecord{Name: "example.com.", Text: text}); err != nil {
			t.Fatalf("failed to add record: %s", err)
		}

		for name, log := range map[string][]landns.Record{"upstream": upstreamLog, "hook": hookLog} {
			if len(log) != i+1 {
				t.Errorf("unexpected %s log length: expected %d but got %d", name, i, len(log))
			} else if tr, ok := log[i].(landns.TxtRecord); !ok {
				t.Errorf("unexpected record type in %s log: %#v", name, log[i])
			} else if tr.Text != text {
				t.Errorf(`unexpected text in %s log: expected "%s" but got "%s"`, name, text, tr.Text)
			}
		}
	}
}

func TestResponseWriterHook_error(t *testing.T) {
	t.Parallel()

	testError := fmt.Errorf("test error")

	hook := landns.ResponseWriterHook{
		Writer: testutil.EmptyResponseWriter{},
		OnAdd: func(r landns.Record) error {
			return testError
		},
	}

	if err := hook.Add(landns.TxtRecord{Name: "example.com.", Text: "hello world"}); err == nil {
		t.Errorf("expected error but got nil")
	} else if err != testError {
		t.Errorf("unexpected error\nexpected: %#v\nbut got: %#v", testError, err)
	}
}

func TestMessageBuilder(t *testing.T) {
	t.Parallel()

	builder := landns.NewMessageBuilder(&dns.Msg{}, true)

	if builder.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: %v", builder.IsAuthoritative())
	}

	if err := builder.Add(landns.AddressRecord{Name: "example.com.", TTL: 42, Address: net.ParseIP("127.0.1.2")}); err != nil {
		t.Errorf("failed to add record: %s", err)
	}

	msg := builder.Build()
	if len(msg.Answer) != 1 {
		t.Errorf("unexpected answer length: expected 1 but got %d", len(msg.Answer))
	} else if msg.Answer[0].String() != "example.com.\t42\tIN\tA\t127.0.1.2" {
		t.Errorf(`unexpected answer: expected "%s" but got "%s"`, "example.com.\t42\tIN\tA\t127.0.1.2", msg.Answer[0].String())
	}
	if msg.Authoritative != true {
		t.Errorf("unexpected authoritative: %v", msg.Authoritative)
	}
	if msg.RecursionAvailable != true {
		t.Errorf("unexpected recurtion available: %v", msg.RecursionAvailable)
	}

	builder.SetNoAuthoritative()
	if err := builder.Add(landns.AddressRecord{Name: "blanktar.jp.", TTL: 1234, Address: net.ParseIP("127.1.2.3")}); err != nil {
		t.Errorf("failed to add record: %s", err)
	}

	msg = builder.Build()
	if len(msg.Answer) != 2 {
		t.Errorf("unexpected answer length: expected 2 but got %d", len(msg.Answer))
	} else {
		for i, expect := range []string{"example.com.\t42\tIN\tA\t127.0.1.2", "blanktar.jp.\t1234\tIN\tA\t127.1.2.3"} {
			if msg.Answer[i].String() != expect {
				t.Errorf(`unexpected answer: expected "%s" but got "%s"`, expect, msg.Answer[i].String())
			}
		}
	}
	if msg.Authoritative != false {
		t.Errorf("unexpected authoritative: %v", msg.RecursionAvailable)
	}
	if msg.RecursionAvailable != true {
		t.Errorf("unexpected recurtion available: %v", msg.RecursionAvailable)
	}

	builder = landns.NewMessageBuilder(&dns.Msg{}, false)
	msg = builder.Build()
	if msg.RecursionAvailable != false {
		t.Errorf("unexpected recurtion available: %v", msg.RecursionAvailable)
	}
}
