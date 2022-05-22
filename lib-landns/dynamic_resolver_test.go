package landns_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

var (
	DynamicResolverTests = []struct {
		Name   string
		Tester func(testing.TB, landns.DynamicResolver)
	}{
		{"SetRecords", DynamicResolverTest_SetRecords},
		{"SetRecords_updateTTL", DynamicResolverTest_SetRecords_updateTTL},
		{"Records", DynamicResolverTest_Records},
		{"GetRecord", DynamicResolverTest_GetRecord},
		{"SearchRecords", DynamicResolverTest_SearchRecords},
		{"GlobRecords", DynamicResolverTest_GlobRecords},
		{"Resolve", DynamicResolverTest_Resolve},
		{"RemoveRecord", DynamicResolverTest_RemoveRecord},
		{"RecursionAvailable", DynamicResolverTest_RecursionAvailable},
		{"volatile", DynamicResolverTest_Volatile},
		{"parallel", func(t testing.TB, r landns.DynamicResolver) {
			ParallelResolveTest(t, r)
		}},
	}
)

func TestDynamicRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input  string
		Expect string
		Error  string
	}{
		{"example.com. 42 IN A 127.0.1.2 ; ID:123", "example.com. 42 IN A 127.0.1.2 ; ID:123", ""},
		{"example.com. 123 IN A 127.3.4.5 ; dummy:aa ID:67 foo:bar", "example.com. 123 IN A 127.3.4.5 ; ID:67", ""},
		{"hello 100 IN A 127.6.7.8", "hello. 100 IN A 127.6.7.8", ""},
		{"v6.example.com. 321 IN AAAA 4::2", "v6.example.com. 321 IN AAAA 4::2", ""},
		{"example.com.\t135\tIN\tTXT\thello\t;\tID:1", "example.com. 135 IN TXT \"hello\" ; ID:1", ""},
		{"c.example.com. IN CNAME example.com. ; ID:2", "c.example.com. 3600 IN CNAME example.com. ; ID:2", ""},
		{"_web._tcp.example.com. SRV 1 2 3 example.com. ; ID:4", "_web._tcp.example.com. 3600 IN SRV 1 2 3 example.com. ; ID:4", ""},
		{"2.1.0.127.in-arpa.addr. 2 IN PTR example.com. ; ID:987654321", "2.1.0.127.in-arpa.addr. 2 IN PTR example.com. ; ID:987654321", ""},
		{"; disabled.com. 100 IN A 127.1.2.3", ";disabled.com. 100 IN A 127.1.2.3", ""},
		{";disabled.com. 100 IN A 127.1.2.3 ; ID:4", ";disabled.com. 100 IN A 127.1.2.3 ; ID:4", ""},
		{"volatile.com. 100 IN A 127.1.2.3 ; Volatile", "volatile.com. 100 IN A 127.1.2.3 ; Volatile", ""},
		{";disabled.volatile.com. 100 IN A 127.1.2.3 ; Id:5 VOLATILE", ";disabled.volatile.com. 100 IN A 127.1.2.3 ; ID:5 Volatile", ""},
		{"a\nb", "", landns.ErrMultiLineDynamicRecord.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID: 42", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID:foobar", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"hello world ; ID:1", "", `failed to parse record: dns: not a TTL: "world" at line: 1:12`},
	}

	for _, tt := range tests {
		r, err := landns.NewDynamicRecord(tt.Input)
		if err != nil && tt.Error == "" {
			t.Errorf("failed to unmarshal dynamic record: %v", err)
			continue
		} else if err != nil && err.Error() != tt.Error {
			t.Errorf(`unmarshal dynamic record: expected error "%v" but got "%v"`, tt.Error, err)
			continue
		}
		if tt.Error != "" {
			continue
		}

		if got, err := r.MarshalText(); err != nil {
			t.Errorf("failed to marshal dynamic record: %v", err)
		} else if string(got) != tt.Expect {
			t.Errorf("encoded text was unexpected:\n\texpected: %#v\n\tbut got:  %#v", tt.Expect, string(got))
		}
	}
}

func TestDynamicRecordSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input  string
		Expect string
		Error  string
	}{
		{"example.com. 42 IN A 127.0.1.2 ; ID:3", "example.com. 42 IN A 127.0.1.2 ; ID:3\n", ""},
		{"example.com. 42 IN A 127.0.1.2 ; ID:3\nexample.com. 24 IN AAAA 1:2:3::4", "example.com. 42 IN A 127.0.1.2 ; ID:3\nexample.com. 24 IN AAAA 1:2:3::4\n", ""},
		{"\n\n\nexample.com. 42 IN A 127.0.1.2 ; ID:3\n\n", "example.com. 42 IN A 127.0.1.2 ; ID:3\n", ""},
		{";this\n  ;is\n\t; comment", "", ""},
		{"unexpected\nexample.com. 1 IN A 127.1.2.3\n\naa", "", "line 1: invalid format: unexpected\nline 4: invalid format: aa"},
	}

	for _, tt := range tests {
		rs, err := landns.NewDynamicRecordSet(tt.Input)
		if err != nil && tt.Error == "" {
			t.Errorf("failed to unmarshal dynamic record set: %v", err)
			continue
		} else if err != nil && err.Error() != tt.Error {
			t.Errorf(`unmarshal dynamic record set: expected error "%v" but got "%v"`, tt.Error, err)
			continue
		}
		if tt.Error != "" {
			continue
		}

		if got, err := rs.MarshalText(); err != nil {
			t.Errorf("failed to marshal dynamic record set: %v", err)
		} else if string(got) != tt.Expect {
			t.Errorf("encoded text was unexpected:\n\texpected: %#v\n\tbut got:  %#v", tt.Expect, string(got))
		}
	}
}

func DynamicResolverTest_SetRecords(t testing.TB, resolver landns.DynamicResolver) {
	tests := []struct {
		Records string
		Expect  []string
	}{
		{
			Records: `
				example.com. 42 IN A 127.0.0.1
				example.com. 100 IN A 127.0.0.2
				example.com. 200 IN AAAA 4::2
				example.com. 300 IN TXT "hello world"
				abc.example.com. 400 IN CNAME example.com.
				abc.example.com. 400 IN CNAME example.com.
				example.com. 500 IN MX 10 mx.example.com.
				example.com. IN NS ns1.example.com.
			`,
			Expect: []string{
				"example.com. 42 IN A 127.0.0.1 ; ID:1",
				"1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
				"example.com. 100 IN A 127.0.0.2 ; ID:3",
				"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				"example.com. 200 IN AAAA 4::2 ; ID:5",
				"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				"example.com. 300 IN TXT \"hello world\" ; ID:7",
				"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				"example.com. IN NS ns1.example.com. ; ID:10",
			},
		},
		{
			Records: `
				;example.com. 42 IN A 127.0.0.1
				new.example.com. 42 IN A 127.0.1.1
				abc.example.com. 400 IN CNAME example.com.
			`,
			Expect: []string{
				"example.com. 100 IN A 127.0.0.2 ; ID:3",
				"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				"example.com. 200 IN AAAA 4::2 ; ID:5",
				"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				"example.com. 300 IN TXT \"hello world\" ; ID:7",
				"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				"example.com. IN NS ns1.example.com. ; ID:10",
				"new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
			},
		},
		{
			Records: `
				;example.com. 42 IN A 127.0.0.1 ; ID:1
				;example.com. 200 IN AAAA 4::2 ; ID:5
				;no.example.com. 200 IN AAAA 4::2 ; ID:5
				;example.com. 500 IN MX 10 mx.example.com.
			`,
			Expect: []string{
				"example.com. 100 IN A 127.0.0.2 ; ID:3",
				"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				"example.com. 300 IN TXT \"hello world\" ; ID:7",
				"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				"example.com. IN NS ns1.example.com. ; ID:10",
				"new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
			},
		},
	}

	for _, tt := range tests {
		records, err := landns.NewDynamicRecordSet(tt.Records)
		if err != nil {
			t.Fatalf("failed to make dynamic records: %s", err)
		}

		if err := resolver.SetRecords(records); err != nil {
			t.Errorf("failed to set records: %s", err)
		}

		rs, err := resolver.Records()
		if err != nil {
			t.Errorf("failed to get records: %s", err)
		}
		AssertDynamicRecordSet(t, tt.Expect, rs)
	}
}

func DynamicResolverTest_SetRecords_updateTTL(t testing.TB, resolver landns.DynamicResolver) {
	tests := []struct {
		Records string
		Expect  []string
	}{
		{
			Records: `
				example.com. 42 IN A 127.0.0.1
			`,
			Expect: []string{
				"example.com. 42 IN A 127.0.0.1 ; ID:1",
				"1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
			},
		},
		{
			Records: `
				example.com. 84 IN A 127.0.0.1
			`,
			Expect: []string{
				"example.com. 84 IN A 127.0.0.1 ; ID:1",
				"1.0.0.127.in-addr.arpa. 84 IN PTR example.com. ; ID:2",
			},
		},
		{
			Records: `
				;example.com. 42 IN A 127.0.0.1 ; ID:1
			`,
			Expect: []string{
				"example.com. 84 IN A 127.0.0.1 ; ID:1",
				"1.0.0.127.in-addr.arpa. 84 IN PTR example.com. ; ID:2",
			},
		},
		{
			Records: `
				;example.com. 84 IN A 127.0.0.1 ; ID:1
			`,
			Expect: []string{},
		},
	}

	for _, tt := range tests {
		records, err := landns.NewDynamicRecordSet(tt.Records)
		if err != nil {
			t.Fatalf("failed to make dynamic records: %s", err)
		}

		if err := resolver.SetRecords(records); err != nil {
			t.Errorf("failed to set records: %s", err)
		}

		rs, err := resolver.Records()
		if err != nil {
			t.Errorf("failed to get records: %s", err)
		}
		AssertDynamicRecordSet(t, tt.Expect, rs)
	}
}

func DynamicResolverTest_Records(t testing.TB, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		example.com. 42 IN A 127.0.0.1
		example.com. 100 IN A 127.0.0.2
		example.com. 200 IN AAAA 4::2
		example.com. 300 IN TXT "hello world"
		abc.example.com. 400 IN CNAME example.com.
		example.com. 500 IN MX 10 mx.example.com.
		example.com. IN NS ns1.example.com.
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	expect := []string{
		"example.com. 42 IN A 127.0.0.1 ; ID:1",
		"1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
		"example.com. 100 IN A 127.0.0.2 ; ID:3",
		"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
		"example.com. 200 IN AAAA 4::2 ; ID:5",
		"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
		"example.com. 300 IN TXT \"hello world\" ; ID:7",
		"abc.example.com. 400 IN CNAME example.com. ; ID:8",
		"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
		"example.com. IN NS ns1.example.com. ; ID:10",
	}

	rs, err := resolver.Records()
	if err != nil {
		t.Errorf("failed to get records: %s", err)
	}
	AssertDynamicRecordSet(t, expect, rs)
}

func DynamicResolverTest_GetRecord(t testing.TB, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		example.com. 42 IN A 127.0.0.1
		example.com. 100 IN A 127.0.0.2
		example.com. 200 IN AAAA 4::2
		example.com. 300 IN TXT "hello world"
		abc.example.com. 400 IN CNAME example.com.
		example.com. 500 IN MX 10 mx.example.com.
		example.com. IN NS ns1.example.com.
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	tests := []struct {
		ID     int
		Expect string
	}{
		{1, "example.com. 42 IN A 127.0.0.1 ; ID:1\n"},
		{20, ""},
	}

	for _, tt := range tests {
		r, err := resolver.GetRecord(tt.ID)
		if err != nil {
			t.Errorf("failed to get record: %s", err)
			continue
		}

		if r.String() != tt.Expect {
			t.Errorf("unexpected record: %d:\nexpected: %#v\nbut got:  %#v", tt.ID, tt.Expect, r.String())
		}
	}
}

func DynamicResolverTest_SearchRecords(t testing.TB, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		example.com. 42 IN A 127.0.0.1
		example.com. 100 IN A 127.0.0.2
		example.com. 200 IN AAAA 4::2
		example.com. 300 IN TXT "hello world"
		abc.example.com. 400 IN CNAME example.com.
		example.com. 500 IN MX 10 mx.example.com.
		example.com. IN NS ns1.example.com.
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	tests := map[landns.Domain][]string{
		"example.com.": {
			"example.com. 42 IN A 127.0.0.1 ; ID:1",
			"example.com. 100 IN A 127.0.0.2 ; ID:3",
			"example.com. 200 IN AAAA 4::2 ; ID:5",
			"example.com. 300 IN TXT \"hello world\" ; ID:7",
			"abc.example.com. 400 IN CNAME example.com. ; ID:8",
			"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
			"example.com. IN NS ns1.example.com. ; ID:10",
		},
		"abc.example.com.": {
			"abc.example.com. 400 IN CNAME example.com. ; ID:8",
		},
		"in-addr.arpa.": {
			"1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
			"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
		},
		"arpa.": {
			"1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
			"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
			"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
		},
	}

	for req, expect := range tests {
		rs, err := resolver.SearchRecords(req)
		if err != nil {
			t.Errorf("failed to search records: %s", err)
			continue
		}

		AssertDynamicRecordSet(t, expect, rs)
	}
}

func DynamicResolverTest_GlobRecords(t testing.TB, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		example.com. 42 IN A 127.0.0.1
		example.com. 100 IN A 127.0.0.2
		example.com. 200 IN AAAA 4::2
		example.com. 300 IN TXT "hello world"
		abc.example.com. 400 IN CNAME example.com.
		example.com. 500 IN MX 10 mx.example.com.
		example.com. IN NS ns1.example.com.
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	tests := map[string][]string{
		"*.example.com.": {
			"abc.example.com. 400 IN CNAME example.com. ; ID:8",
		},
		"abc.*": {
			"abc.example.com. 400 IN CNAME example.com. ; ID:8",
		},
		"2*arpa.": {
			"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
			"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
		},
	}

	for req, expect := range tests {
		rs, err := resolver.GlobRecords(req)
		if err != nil {
			t.Errorf("failed to glob records: %s", err)
			continue
		}

		AssertDynamicRecordSet(t, expect, rs)
	}
}

func DynamicResolverTest_Resolve(t testing.TB, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		example.com. 42 IN A 127.0.0.1
		example.com. 100 IN A 127.0.0.2
		example.com. 200 IN AAAA 4::2
		example.com. 300 IN TXT "hello world"
		127.0.0.1.in-addr.arpa. 400 IN PTR example.com.
		abc.example.com. 500 IN CNAME example.com.
		_web._tcp.example.com. 600 IN SRV 1 2 3 example.com.
		example.com. 700 IN MX 10 mx.example.com.
		example.com. IN NS ns1.example.com.
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeA, false),
		true,
		"example.com. 42 IN A 127.0.0.1",
		"example.com. 100 IN A 127.0.0.2",
	)

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeAAAA, false),
		true,
		"example.com. 200 IN AAAA 4::2",
	)

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeTXT, false),
		true,
		"example.com. 300 IN TXT \"hello world\"",
	)

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("127.0.0.1.in-addr.arpa.", dns.TypePTR, false),
		true,
		"127.0.0.1.in-addr.arpa. 400 IN PTR example.com.",
	)

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("abc.example.com.", dns.TypeCNAME, false),
		true,
		"abc.example.com. 500 IN CNAME example.com.",
	)

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("_web._tcp.example.com.", dns.TypeSRV, false),
		true,
		"_web._tcp.example.com. 600 IN SRV 1 2 3 example.com.",
	)

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeMX, false),
		true,
		"example.com. 700 IN MX 10 mx.example.com.",
	)

	AssertResolve(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeNS, false),
		true,
		"example.com. IN NS ns1.example.com.",
	)
}

func DynamicResolverTest_RemoveRecord(t testing.TB, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		example.com. 42 IN A 127.0.0.1
		example.com. 100 IN A 127.0.0.2
		example.com. 200 IN AAAA 4::2
		example.com. 300 IN TXT "hello world"
		abc.example.com. 400 IN CNAME example.com.
		example.com. 500 IN MX 10 mx.example.com.
		example.com. IN NS ns1.example.com.
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	tests := []struct {
		Entries map[int]string
		Target  int
		Error   error
	}{
		{
			Entries: map[int]string{
				1:  "example.com. 42 IN A 127.0.0.1 ; ID:1",
				2:  "1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
				3:  "example.com. 100 IN A 127.0.0.2 ; ID:3",
				4:  "2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				5:  "example.com. 200 IN AAAA 4::2 ; ID:5",
				6:  "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				7:  "example.com. 300 IN TXT \"hello world\" ; ID:7",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
			},
			Target: 2,
			Error:  nil,
		},
		{
			Entries: map[int]string{
				1:  "example.com. 42 IN A 127.0.0.1 ; ID:1",
				3:  "example.com. 100 IN A 127.0.0.2 ; ID:3",
				4:  "2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				5:  "example.com. 200 IN AAAA 4::2 ; ID:5",
				6:  "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				7:  "example.com. 300 IN TXT \"hello world\" ; ID:7",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
			},
			Target: 7,
			Error:  nil,
		},
		{
			Entries: map[int]string{
				1:  "example.com. 42 IN A 127.0.0.1 ; ID:1",
				5:  "example.com. 200 IN AAAA 4::2 ; ID:5",
				6:  "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
			},
			Target: 3,
			Error:  nil,
		},
		{
			Entries: map[int]string{
				1:  "example.com. 42 IN A 127.0.0.1 ; ID:1",
				5:  "example.com. 200 IN AAAA 4::2 ; ID:5",
				6:  "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
			},
			Target: 2,
			Error:  landns.ErrNoSuchRecord,
		},
		{
			Entries: map[int]string{
				1:  "example.com. 42 IN A 127.0.0.1 ; ID:1",
				5:  "example.com. 200 IN AAAA 4::2 ; ID:5",
				6:  "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
			},
			Target: 2,
			Error:  landns.ErrNoSuchRecord,
		},
	}

	for _, tt := range tests {
		for id, expect := range tt.Entries {
			record, err := resolver.GetRecord(id)
			if err != nil {
				t.Errorf("failed to get record: %d: %d", id, err)
				continue
			}

			if record.String() != expect+"\n" {
				t.Errorf("failed to get record: %d:\nexpected: %#v\nbut got:  %#v", id, expect+"\n", record.String())
			}
		}

		if err := resolver.RemoveRecord(tt.Target); err != tt.Error {
			if tt.Error == nil {
				t.Errorf("failed to remove record: %s", err)
			} else {
				t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", tt.Error, err)
			}
		}
	}
}

func DynamicResolverTest_Volatile(t testing.TB, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		fixed.example.com. 100 IN TXT "fixed"
		long.example.com. 100 IN TXT "long" ; Volatile
		short.example.com. 1 IN TXT "short" ; Volatile
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	time.Sleep(1500 * time.Millisecond)

	rs, err := resolver.Records()
	if err != nil {
		t.Errorf("failed to get records: %s", err)
	}
	AssertDynamicRecordSet(t, []string{
		`fixed.example.com. 100 IN TXT "fixed" ; ID:1`,
		`long.example.com. 98 IN TXT "long" ; ID:2 Volatile`,
	}, rs)

	AssertResolve(t, resolver, landns.NewRequest("long.example.com.", dns.TypeTXT, false), true, `long.example.com. 98 IN TXT "long"`)
}

func DynamicResolverTest_RecursionAvailable(t testing.TB, resolver landns.DynamicResolver) {
	if resolver.RecursionAvailable() != false {
		t.Errorf("unexpected recursion available value: expected false but got true")
	}
}

func DynamicResolverBenchmark(b *testing.B, resolver landns.DynamicResolver) {
	records := make(landns.DynamicRecordSet, 200)

	var err error
	for i := 0; i < 100; i++ {
		records[i*2], err = landns.NewDynamicRecord(fmt.Sprintf("host%d.example.com. 0 IN A 127.1.2.3", i))
		if err != nil {
			b.Fatalf("failed to make dynamic record: %v", err)
		}

		records[i*2+1], err = landns.NewDynamicRecord(fmt.Sprintf("host%d.example.com. 0 IN A 127.2.3.4", i))
		if err != nil {
			b.Fatalf("failed to make dynamic record: %v", err)
		}
	}

	if err := resolver.SetRecords(records); err != nil {
		b.Fatalf("failed to set records: %s", err)
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(testutil.NewDummyResponseWriter(), req)
	}

	b.StopTimer()
}

func ExampleDynamicRecord() {
	record, _ := landns.NewDynamicRecord("example.com. 600 IN A 127.0.0.1")
	fmt.Println("name:", record.Record.GetName(), "disabled:", record.Disabled)

	record, _ = landns.NewDynamicRecord(";test.service 300 IN TXT \"hello world\"")
	fmt.Println("name:", record.Record.GetName(), "disabled:", record.Disabled)

	// Output:
	// name: example.com. disabled: false
	// name: test.service. disabled: true
}

func ExampleDynamicRecord_String() {
	record, _ := landns.NewDynamicRecord("example.com. 600 IN A 127.0.0.1")

	fmt.Println(record)

	record.Disabled = true
	fmt.Println(record)

	id := 10
	record.ID = &id
	fmt.Println(record)

	// Output:
	// example.com. 600 IN A 127.0.0.1
	// ;example.com. 600 IN A 127.0.0.1
	// ;example.com. 600 IN A 127.0.0.1 ; ID:10
}

func ExampleDynamicRecordSet() {
	records, _ := landns.NewDynamicRecordSet(`
	a.example.com. 100 IN A 127.0.0.1
	b.example.com. 200 IN A 127.0.1.2
`)

	for _, r := range records {
		fmt.Println(r.Record.GetName())
		fmt.Println(r)
	}

	// Output:
	// a.example.com.
	// a.example.com. 100 IN A 127.0.0.1
	// b.example.com.
	// b.example.com. 200 IN A 127.0.1.2
}
