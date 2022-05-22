package landns

import (
	"testing"
)

func TestCompileGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Glob string
		Str  string
		Exp  bool
	}{
		{`ab*def`, `abcdef`, true},
		{`ab.*f`, `abcdef`, false},
		{`ab.*f`, `ab.cdef`, true},
		{`[0-9]*(a+b*)`, `[0-9]---(a+b=====)`, true},
		{`^abc$`, `^abc$`, true},
		{`\.*`, `\.abc`, true},
		{`cd`, `abcdef`, false},
		{`*cd*`, `abcdef`, true},
		{`*.example.com.`, `abc.example.com.`, true},
		{`*.example.com.`, `.example.com.`, true},
		{`*.example.com.`, `abc*example*com*`, false},
	}

	for _, tt := range tests {
		glob, err := compileGlob(tt.Glob)
		if err != nil {
			t.Errorf("failed to compile glob: %#v: %s", tt.Glob, err)
			continue
		}

		if glob(tt.Str) != tt.Exp {
			t.Errorf("failed to test glob: %#v <- %#v: expected %v but got %v", tt.Glob, tt.Str, tt.Exp, glob(tt.Str))
		}
	}
}

func TestEtcdResolver_getIDbyKey(t *testing.T) {
	tests := []struct {
		Key    string
		Expect int
		Error  string
	}{
		{"/landns/com/example/42", 42, ""},
		{"1", 1, ""},
		{"/path/to/somewhere", 0, `failed to parse record ID: strconv.Atoi: parsing "somewhere": invalid syntax`},
		{"", 0, `failed to parse record ID: strconv.Atoi: parsing "": invalid syntax`},
	}

	r := new(EtcdResolver)

	for _, tt := range tests {
		i, err := r.getIDbyKey([]byte(tt.Key))
		if err != nil && tt.Error == "" {
			t.Errorf("%s: unexpected error: %s", tt.Key, err)
			continue
		}
		if tt.Error != "" && (err == nil || err.Error() != tt.Error) {
			t.Errorf("%s: unexpected error:\nexpected: %v\nbut got:  %v", tt.Key, tt.Error, err)
			continue
		}

		if i != tt.Expect {
			t.Errorf("%s: unexpected ID: expected %d but got %d", tt.Key, tt.Expect, i)
		}
	}
}

func TestEtcdResolver_getKey(t *testing.T) {
	id := int(42)

	tests := []struct {
		Record DynamicRecord
		Expect string
	}{
		{DynamicRecord{Record: TxtRecord{Name: "example.com.", TTL: 10, Text: "hello world"}, ID: nil}, "/landns_test/records/com/example"},
		{DynamicRecord{Record: TxtRecord{Name: "example.com.", TTL: 10, Text: "hello world"}, ID: &id}, "/landns_test/records/com/example/42"},
	}

	r := &EtcdResolver{
		Prefix: "/landns_test",
	}

	for _, tt := range tests {
		s := r.getKey(tt.Record)
		if s != tt.Expect {
			t.Errorf("%s: unexpected ID:\nexpected: %s\nbut got:  %s", tt.Record, tt.Expect, s)
		}
	}
}
