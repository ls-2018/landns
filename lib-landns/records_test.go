package landns_test

import (
	"fmt"
	"net"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func TestDomain_Validate(t *testing.T) {
	t.Parallel()

	a := landns.Domain("")
	if err := a.Validate(); err == nil {
		t.Errorf("failed to empty domain validation: <nil>")
	} else if err.Error() != `invalid domain: ""` {
		t.Errorf("failed to empty domain validation: %#v", err.Error())
	}

	b := landns.Domain("..")
	if err := b.Validate(); err == nil {
		t.Errorf("failed to invalid domain validation: <nil>")
	} else if err.Error() != `invalid domain: ".."` {
		t.Errorf("failed to invalid domain validation: %#v", err.Error())
	}

	c := landns.Domain("example.com.")
	if err := c.Validate(); err != nil {
		t.Errorf("failed to valid domain validation: %#v", err.Error())
	}

	d := landns.Domain("example.com")
	if err := d.Validate(); err != nil {
		t.Errorf("failed to valid domain validation: %#v", err.Error())
	}
}

func TestDomain_Encoding(t *testing.T) {
	t.Parallel()

	var d landns.Domain

	for input, expect := range map[string]string{"": ".", "example.com": "example.com.", "blanktar.jp.": "blanktar.jp."} {
		if err := (&d).UnmarshalText([]byte(input)); err != nil {
			t.Errorf("failed to unmarshal: %s: %s", input, err)
		} else if result, err := d.MarshalText(); err != nil {
			t.Errorf("failed to marshal: %s: %s", input, err)
		} else if string(result) != expect {
			t.Errorf("unexpected marshal result: expected %s but got %s", expect, string(result))
		}
	}

	if err := (&d).UnmarshalText([]byte("example.com..")); err == nil {
		t.Errorf("expected error but got nil")
	} else if err.Error() != `invalid domain: "example.com.."` {
		t.Errorf(`unexpected error: expected 'invalid domain: "example.com.."' but got '%s'`, err)
	}
}

func TestDomain_ToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input  landns.Domain
		Expect string
	}{
		{"example.com.", "/com/example"},
		{"", "/"},
		{"a.b.c.d", "/d/c/b/a"},
	}

	for _, tt := range tests {
		if p := tt.Input.ToPath(); p != tt.Expect {
			t.Errorf("unexpected path:\nexpected: %s\nbut got:  %s", tt.Expect, p)
		}
	}
}

func TestNewRecordWithTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		String string
		TTL    uint32
		Expect string
		Error  string
	}{
		{"example.com. 600 IN A 127.0.0.1", 42, "example.com. 42 IN A 127.0.0.1", ""},
		{"example.com. 500 IN A 127.0.0.2", 42, "example.com. 42 IN A 127.0.0.2", ""},
		{"example.com. 400 IN A 127.0.0.3", 400, "example.com. 400 IN A 127.0.0.3", ""},
		{"example.com. 300 IN A 127.0.0.3", 0, "example.com. 0 IN A 127.0.0.3", ""},
		{"hello world", 42, "", `failed to parse record: .+`},
	}

	for _, tt := range tests {
		r, err := landns.NewRecordWithTTL(tt.String, tt.TTL)

		if err != nil {
			if tt.Error == "" {
				t.Errorf("failed to parse record: %s", err)
			} else if ok, e := regexp.MatchString("^"+tt.Error+"$", err.Error()); e != nil || !ok {
				t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", tt.Error, err.Error())
			}
			continue
		}

		if r.String() != tt.Expect {
			t.Errorf("unexpected parse result:\nexpected: %#v\nbut got:  %#v", tt.Expect, r.String())
		}
	}
}

func TestNewRecordWithExpire(t *testing.T) {
	t.Parallel()

	tests := []struct {
		String string
		Offset time.Duration
		Expect string
		Error  string
	}{
		{"example.com. 600 IN A 127.0.0.1", 42 * time.Second, "example.com. 42 IN A 127.0.0.1", ""},
		{"example.com. 500 IN A 127.0.0.2", 42 * time.Second, "example.com. 42 IN A 127.0.0.2", ""},
		{"example.com. 400 IN A 127.0.0.3", 400 * time.Second, "example.com. 400 IN A 127.0.0.3", ""},
		{"example.com. 300 IN A 127.0.0.3", time.Millisecond, "example.com. 0 IN A 127.0.0.3", ""},
		{"example.com. 400 IN A 127.0.0.3", -time.Second, "", `expire can't be past time: 20..-..-.. ..:..:..\.[0-9]+ .*`},
	}

	for _, tt := range tests {
		r, err := landns.NewRecordWithExpire(tt.String, time.Now().Add(tt.Offset))

		if err != nil {
			if tt.Error == "" {
				t.Errorf("failed to parse record: %s", err)
			} else if ok, e := regexp.MatchString("^"+tt.Error+"$", err.Error()); e != nil || !ok {
				t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", tt.Error, err.Error())
			}
			continue
		}

		if r.String() != tt.Expect {
			t.Errorf("unexpected parse result:\nexpected: %#v\nbut got:  %#v", tt.Expect, r.String())
		}
	}
}

func TestRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Record     landns.Record
		WithTTL    string
		WithoutTTL string
		Qtype      uint16
		TTL        uint32
	}{
		{
			landns.AddressRecord{Name: "a.example.com.", TTL: 10, Address: net.ParseIP("127.0.0.1")},
			"a.example.com. 10 IN A 127.0.0.1",
			"a.example.com. 0 IN A 127.0.0.1",
			dns.TypeA,
			10,
		},
		{
			landns.AddressRecord{Name: "aaaa.example.com.", TTL: 20, Address: net.ParseIP("4::2")},
			"aaaa.example.com. 20 IN AAAA 4::2",
			"aaaa.example.com. 0 IN AAAA 4::2",
			dns.TypeAAAA,
			20,
		},
		{
			landns.NsRecord{Name: "ns.example.com.", Target: "example.com."},
			"ns.example.com. IN NS example.com.",
			"ns.example.com. IN NS example.com.",
			dns.TypeNS,
			0,
		},
		{
			landns.CnameRecord{Name: "cname.example.com.", TTL: 40, Target: "example.com."},
			"cname.example.com. 40 IN CNAME example.com.",
			"cname.example.com. 0 IN CNAME example.com.",
			dns.TypeCNAME,
			40,
		},
		{
			landns.PtrRecord{Name: "1.0.0.127.in-addr.arpa.", TTL: 50, Domain: "ptr.example.com."},
			"1.0.0.127.in-addr.arpa. 50 IN PTR ptr.example.com.",
			"1.0.0.127.in-addr.arpa. 0 IN PTR ptr.example.com.",
			dns.TypePTR,
			50,
		},
		{
			landns.MxRecord{Name: "mx.example.com.", TTL: 60, Preference: 42, Target: "example.com."},
			"mx.example.com. 60 IN MX 42 example.com.",
			"mx.example.com. 0 IN MX 42 example.com.",
			dns.TypeMX,
			60,
		},
		{
			landns.TxtRecord{Name: "txt.example.com.", TTL: 70, Text: "hello world"},
			"txt.example.com. 70 IN TXT \"hello world\"",
			"txt.example.com. 0 IN TXT \"hello world\"",
			dns.TypeTXT,
			70,
		},
		{
			landns.SrvRecord{Name: "_web._tcp.srv.example.com.", TTL: 80, Priority: 11, Weight: 22, Port: 33, Target: "example.com."},
			"_web._tcp.srv.example.com. 80 IN SRV 11 22 33 example.com.",
			"_web._tcp.srv.example.com. 0 IN SRV 11 22 33 example.com.",
			dns.TypeSRV,
			80,
		},
	}

	for _, tt := range tests {
		if err := tt.Record.Validate(); err != nil {
			t.Errorf("failed to validate: %s", err)
		}
		if s := tt.Record.String(); s != tt.WithTTL {
			t.Errorf("failed to convert to string with TTL:\nexpected: %s\nbut got:  %s", tt.WithTTL, s)
		}
		if s := tt.Record.WithoutTTL(); s != tt.WithoutTTL {
			t.Errorf("failed to convert to string without TTL:\nexpected: %s\nbut got:  %s", tt.WithoutTTL, s)
		}
		if q := tt.Record.GetQtype(); q != tt.Qtype {
			t.Errorf("unexpected qtype: expected %d but got %d", tt.Qtype, q)
		}
		if ttl := tt.Record.GetTTL(); ttl != tt.TTL {
			t.Errorf("unexpected ttl: expected %d but got %d", tt.TTL, ttl)
		}

		rr1, err := tt.Record.ToRR()
		if err != nil {
			t.Errorf("failed to convert to dns.RR: %s", err)
			continue
		}

		rr2, err := dns.NewRR(tt.WithTTL)
		if err != nil {
			t.Errorf("failed to convert example to dns.RR: %s", err)
			continue
		}

		if rr1.String() != rr2.String() {
			t.Errorf("unexpected RR:\nexpected: %s\nbut got:  %s", rr2, rr1)
		}
	}
}

func TestVolatileRecord(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		tests := []struct {
			Entry  string
			Expect string
		}{
			{
				fmt.Sprintf("example.com. 600 IN A 127.0.0.1 ; %d", time.Now().Add(42500*time.Millisecond).Unix()),
				"example.com. 42 IN A 127.0.0.1",
			},
			{
				fmt.Sprintf("example.com. 600 IN A 127.0.0.1;%d", time.Now().Add(42500*time.Millisecond).Unix()),
				"example.com. 42 IN A 127.0.0.1",
			},
			{
				"example.com. 600 IN A 127.0.0.1",
				"example.com. 600 IN A 127.0.0.1",
			},
		}

		for _, tt := range tests {
			e, err := landns.NewVolatileRecord(tt.Entry)

			if err != nil {
				t.Errorf("%#v: failed to parse cache entry: %s", tt.Entry, err)
				continue
			}

			if r, err := e.Record(); err != nil {
				t.Errorf("%#v: failed to get record: %s", tt.Entry, err)
			} else if r.String() != tt.Expect {
				t.Errorf("%#v: unexpected record string:\nexpected: %#v\nbut got:  %#v", tt.Entry, tt.Expect, r.String())
			}
		}
	})

	t.Run("error", func(t *testing.T) {
		tests := []struct {
			Entry string
			Error string
		}{
			{
				"example.com. 600 IN A 127.0.0.1 ; 12345",
				"failed to parse record: expire can't be past time: 1970-01-01 ..:..:.. [+-].... ...",
			},
			{
				"hello world ; 4294967295",
				"failed to parse record: dns: not a TTL: \"world\" at line: 1:12",
			},
			{
				"example.com. 600 IN A 127.0.0.1 ; ",
				"failed to parse record: strconv\\.ParseInt: parsing \"\": invalid syntax",
			},
		}

		for _, tt := range tests {
			_, err := landns.NewVolatileRecord(tt.Entry)

			if err == nil {
				t.Errorf("%#v: expected error but got nil", tt.Entry)
			} else if ok, e := regexp.MatchString("^"+tt.Error+"$", err.Error()); e != nil || !ok {
				t.Errorf("%#v: unexpected error:\nexpected: %#v\nbut got:  %#v", tt.Entry, tt.Error, err.Error())
			}
		}
	})
}

func ExampleDomain() {
	a := landns.Domain("example.com")
	b := a.Normalized()
	fmt.Println(string(a), "->", string(b))

	c := landns.Domain("")
	d := c.Normalized()
	fmt.Println(string(c), "->", string(d))

	// Output:
	// example.com -> example.com.
	//  -> .
}

func ExampleNewRecord() {
	record, _ := landns.NewRecord("example.com. 600 IN A 127.0.0.1")

	fmt.Println(record.GetName())
	fmt.Println(record.GetTTL())
	fmt.Println(record.String())

	// Output:
	// example.com.
	// 600
	// example.com. 600 IN A 127.0.0.1
}

func ExampleNewRecordWithExpire() {
	record, _ := landns.NewRecordWithExpire("example.com. 600 IN A 127.0.0.1", time.Now().Add(10*time.Second))

	fmt.Println(record.GetName())
	fmt.Println(record.GetTTL())
	fmt.Println(record.String())

	// Output:
	// example.com.
	// 10
	// example.com. 10 IN A 127.0.0.1
}

func ExampleNewRecordFromRR() {
	rr, _ := dns.NewRR("example.com. 600 IN A 127.0.0.1")
	record, _ := landns.NewRecordFromRR(rr)

	fmt.Println(record.GetName())
	fmt.Println(record.GetTTL())
	fmt.Println(record.String())

	// Output:
	// example.com.
	// 600
	// example.com. 600 IN A 127.0.0.1
}
