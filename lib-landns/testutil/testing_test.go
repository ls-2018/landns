package testutil_test

import (
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns/testutil"
)

func TestDummyTB(t *testing.T) {
	t.Parallel()

	type Failed func() bool
	type Logger func(string, ...interface{})
	type Assert func(testutil.SimpleTB, ...string)
	type TestMaker func(tb *testutil.DummyTB) (failed Failed, logger Logger, assert Assert, anotherAssert Assert)

	tests := []struct {
		Name  string
		Maker TestMaker
	}{
		{"Errorf", func(tb *testutil.DummyTB) (Failed, Logger, Assert, Assert) {
			return tb.Failed, tb.Errorf, tb.AssertErrors, tb.AssertFatals
		}},
		{"Fatalf", func(tb *testutil.DummyTB) (Failed, Logger, Assert, Assert) {
			return tb.Failed, tb.Fatalf, tb.AssertFatals, tb.AssertErrors
		}},
	}

	for _, tt := range tests {
		failed, logger, assert, anotherAssert := tt.Maker(new(testutil.DummyTB))

		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			if failed() {
				t.Errorf("unexpected failed state: failed")
			}

			logger("this is dummy error: %s: %d", "some value", 123)

			assert(t, "this is dummy error: some value: 123")
			assert(t, "this is dummy error: ???? value: [0-9]*")
			anotherAssert(t)

			tests := []struct {
				Assertion []string
				Expect    []string
			}{
				{
					[]string{"this is dummy error: some value: 123", "hello world"},
					[]string{strings.Join([]string{
						"unexpected errors log:",
						"expected:",
						"\t\"this is dummy error: some value: 123\"",
						"\t\"hello world\"",
						"but got:",
						"\t\"this is dummy error: some value: 123\"",
						"",
					}, "\n")},
				},
				{
					[]string{"this is dummy error: some value: 124"},
					[]string{strings.Join([]string{
						"unexpected errors log:",
						"expected:",
						"\t\"this is dummy error: some value: 124\"",
						"but got:",
						"\t\"this is dummy error: some value: 123\"",
						"",
					}, "\n")},
				},
				{
					[]string{"this is dummy error: some value: [123"},
					[]string{"failed to compare log entry: syntax error in pattern"},
				},
			}

			for _, tt := range tests {
				tb2 := new(testutil.DummyTB)
				assert(tb2, tt.Assertion...)
				tb2.AssertErrors(t, tt.Expect...)
			}

			if !failed() {
				t.Errorf("unexpected failed state: not failed")
			}
		})
	}
}

func TestDummyTB_Fatalf(t *testing.T) {
	t.Parallel()

	tb := new(testutil.DummyTB)

	if tb.Failed() {
		t.Errorf("unexpected failed state: failed")
	}

	tb.Fatalf("this is dummy error: %s: %d", "some value", 123)

	tb.AssertFatals(t, "this is dummy error: some value: 123")
	tb.AssertFatals(t, "this is dummy error: ???? value: *")
	tb.AssertErrors(t)

	if !tb.Failed() {
		t.Errorf("unexpected failed state: not failed")
	}
}
