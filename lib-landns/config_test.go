package landns_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns"
)

func TestProto_Validate(t *testing.T) {
	t.Parallel()

	a := landns.Proto("")
	if err := a.Validate(); err != nil {
		t.Errorf("failed to empty proto validation: %#v", err.Error())
	}

	b := landns.Proto("foobar")
	if err := b.Validate(); err == nil {
		t.Errorf("failed to invalid proto validation: <nil>")
	} else if err.Error() != `invalid protocol: foobar` {
		t.Errorf("failed to invalid proto validation: %#v", err.Error())
	}

	c := landns.Proto("tcp")
	if err := c.Validate(); err != nil {
		t.Errorf("failed to tcp proto validation: %#v", err.Error())
	}

	d := landns.Proto("udp")
	if err := d.Validate(); err != nil {
		t.Errorf("failed to udp proto validation: %#v", err.Error())
	}
}

func TestProto_Encoding(t *testing.T) {
	t.Parallel()

	var p landns.Proto

	for input, expect := range map[string]string{"tcp": "tcp", "udp": "udp", "": "tcp"} {
		if err := (&p).UnmarshalText([]byte(input)); err != nil {
			t.Errorf("failed to unmarshal: %s: %s", input, err)
		} else if result, err := p.MarshalText(); err != nil {
			t.Errorf("failed to marshal: %s: %s", input, err)
		} else if string(result) != expect {
			t.Errorf("unexpected marshal result: expected %s but got %s", expect, string(result))
		}
	}

	if err := (&p).UnmarshalText([]byte("foo")); err == nil {
		t.Errorf("expected error but got nil")
	} else if err.Error() != `invalid protocol: foo` {
		t.Errorf(`unexpected error: expected 'invalid protocol: foo' but got '%s'`, err)
	}
}
