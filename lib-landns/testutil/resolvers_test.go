package testutil_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestDummyResolver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err bool
		rec bool
	}{
		{false, false},
		{false, true},
		{true, false},
		{true, true},
	}

	w := testutil.NewDummyResponseWriter()
	r := landns.NewRequest("example.com.", dns.TypeA, true)

	for _, tt := range tests {
		res := testutil.DummyResolver{Error: tt.err, Recursion: tt.rec}

		if err := res.Resolve(w, r); !tt.err && err != nil {
			t.Errorf("unexpected error: %#v", err)
		} else if tt.err && err == nil {
			t.Errorf("expected error but not occurred")
		}

		if res.RecursionAvailable() != tt.rec {
			t.Errorf("unexpected recursion available: expected %#v but got %#v", tt.rec, res.RecursionAvailable())
		}

		if err := res.Close(); err != nil {
			t.Errorf("unexpected error: %#v", err)
		}
	}
}
