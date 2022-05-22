package httplog

import (
	"testing"

	"github.com/macrat/landns/lib-landns/logger/logtest"
)

func TestHTTPLogger_getLogger(t *testing.T) {
	lt := logtest.Start()
	defer lt.Close()

	specific := new(logtest.DummyLogger)

	if l := (HTTPLogger{Logger: specific}.getLogger()); l != specific {
		t.Errorf("unexpected logger: expected %#v (specific one) but got %#v", specific, l)
	}

	if l := (HTTPLogger{}.getLogger()); l != lt.Logger {
		t.Errorf("unexpected logger: expected %#v (globals one) but got %#v", lt.Logger, l)
	}
}
