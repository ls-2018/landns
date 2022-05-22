package testutil

import (
	"fmt"
	"path/filepath"
)

// SimpleTB is the interface to testing.T, testing.B and DummyTB.
type SimpleTB interface {
	Helper()
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Failed() bool
}

// DummyTB is the tester for assert functions.
type DummyTB struct {
	Errors []string
	Fatals []string
}

// Helper is dummy function.
func (t *DummyTB) Helper() {
}

// Errorf is error message recorder.
func (t *DummyTB) Errorf(format string, data ...interface{}) {
	t.Errors = append(t.Errors, fmt.Sprintf(format, data...))
}

// Fatalf is fatal error message recorder.
func (t *DummyTB) Fatalf(format string, data ...interface{}) {
	t.Fatals = append(t.Fatals, fmt.Sprintf(format, data...))
}

// Failed is reports test has failed.
func (t *DummyTB) Failed() bool {
	return len(t.Errors) > 0 || len(t.Fatals) > 0
}

func (t *DummyTB) assertLog(tb SimpleTB, expect []string, got []string) {
	tb.Helper()

	ok := true

	if len(expect) != len(got) {
		ok = false
	} else {
		for i := range expect {
			if m, err := filepath.Match(expect[i], got[i]); err != nil {
				tb.Errorf("failed to compare log entry: %s", err)
				return
			} else if !m {
				ok = false
				break
			}
		}
	}

	if !ok {
		msg := "unexpected errors log:\nexpected:\n"
		for _, e := range expect {
			msg += fmt.Sprintf("\t%#v\n", e)
		}
		msg += "but got:\n"
		for _, g := range got {
			msg += fmt.Sprintf("\t%#v\n", g)
		}
		tb.Errorf(msg)
	}
}

// AssertErrors is test error logs.
func (t *DummyTB) AssertErrors(tb SimpleTB, expect ...string) {
	tb.Helper()

	t.assertLog(tb, expect, t.Errors)
}

// AssertFatals is test fatal logs.
func (t *DummyTB) AssertFatals(tb SimpleTB, expect ...string) {
	tb.Helper()

	t.assertLog(tb, expect, t.Fatals)
}
