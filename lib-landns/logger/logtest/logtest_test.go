package logtest_test

import (
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns/logger"
	"github.com/macrat/landns/lib-landns/logger/logtest"
)

func TestEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Entry  logtest.Entry
		Expect string
	}{
		{
			Entry:  logtest.Entry{logger.ErrorLevel, "hello world", nil},
			Expect: "[error]hello world[]",
		},
		{
			Entry: logtest.Entry{logger.DebugLevel, "foo bar", logger.Fields{
				"id":     1,
				"say":    "hello",
				"target": "world",
			}},
			Expect: "[debug]foo bar[id:1 say:hello target:world]",
		},
	}

	for _, tt := range tests {
		if tt.Entry.String() != tt.Expect {
			t.Errorf("failed to convert to string:\nexpected: %#v\nbut got:  %#v", tt.Expect, tt.Entry.String())
		}
	}
}

func TestDummyLogger(t *testing.T) {
	t.Parallel()

	l := new(logtest.DummyLogger)
	es := []logtest.Entry{}

	if l.GetLevel() != logger.DebugLevel {
		t.Errorf("unexpected level: expected %s but got %s", logger.DebugLevel, l.GetLevel())
	}

	tests := []struct {
		Message string
		Fields  logger.Fields
		Write   func(string, logger.Fields)
		Level   logger.Level
	}{
		{"hello", logger.Fields{"id": 1}, l.Debug, logger.DebugLevel},
		{"world", logger.Fields{"id": 2}, l.Info, logger.InfoLevel},
		{"foo", logger.Fields{"id": 3}, l.Warn, logger.WarnLevel},
		{"bar", logger.Fields{"id": 4}, l.Error, logger.ErrorLevel},
		{"baz", logger.Fields{"id": 5}, l.Fatal, logger.FatalLevel},
	}

	for _, tt := range tests {
		tt.Write(tt.Message, tt.Fields)

		e := logtest.Entry{tt.Level, tt.Message, tt.Fields}

		if err := l.Test([]logtest.Entry{e}); err != nil {
			t.Errorf("%s", err)
		}

		es = append(es, e)
		if err := l.TestAll(es); err != nil {
			t.Errorf("%s", err)
		}
	}
}

func TestDummyLogger_String(t *testing.T) {
	tests := []struct {
		Logger *logtest.DummyLogger
		Expect string
	}{
		{
			&logtest.DummyLogger{
				{logger.DebugLevel, "hello", nil},
				{logger.InfoLevel, "world", logger.Fields{"id": 42}},
			},
			"[debug]hello[]\n[info]world[id:42]",
		},
		{
			&logtest.DummyLogger{
				{logger.DebugLevel, "hello", nil},
			},
			"[debug]hello[]",
		},
		{
			&logtest.DummyLogger{},
			"",
		},
	}

	for _, tt := range tests {
		if tt.Logger.String() != tt.Expect {
			t.Errorf("unexpected string:\nexpected:\n%s\n\nbut got:\n%s", tt.Expect, tt.Logger.String())
		}
	}
}

func TestDummyLogger_Test(t *testing.T) {
	t.Parallel()

	l := new(logtest.DummyLogger)
	l.Debug("hello", logger.Fields{"target": "world"})

	if err := l.Test([]logtest.Entry{{logger.DebugLevel, "hello", logger.Fields{"target": "world"}}}); err != nil {
		t.Error(err)
	}

	expect := strings.Join([]string{
		"unexpected log entries:",
		"expected:",
		"\t[debug]hello[target:world]",
		"\t[warning]world[id:42]",
		"but got:",
		"\t[debug]hello[target:world]",
		"",
	}, "\n")

	err := l.Test([]logtest.Entry{
		{logger.DebugLevel, "hello", logger.Fields{"target": "world"}},
		{logger.WarnLevel, "world", logger.Fields{"id": 42}},
	})

	if err == nil {
		t.Errorf("expected error but got nil")
	} else if err.Error() != expect {
		t.Errorf("unexpected error:\nexpected:\n```\n%s```\n\nbut got:\n```\n%s```", expect, err.Error())
	}

	expect = strings.Join([]string{
		"unexpected log entries:",
		"expected:",
		"\t[warning]world[id:42]",
		"but got:",
		"\t[debug]hello[target:world]",
		"",
	}, "\n")

	err = l.Test([]logtest.Entry{
		{logger.WarnLevel, "world", logger.Fields{"id": 42}},
	})

	if err == nil {
		t.Errorf("expected error but got nil")
	} else if err.Error() != expect {
		t.Errorf("unexpected error:\nexpected:\n```\n%s```\n\nbut got:\n```\n%s```", expect, err.Error())
	}
}

func TestDummyLogger_TestAll(t *testing.T) {
	t.Parallel()

	l := new(logtest.DummyLogger)
	l.Debug("hello", logger.Fields{"target": "world"})
	l.Debug("world", logger.Fields{"say": "hello"})

	if err := l.TestAll([]logtest.Entry{
		{logger.DebugLevel, "hello", logger.Fields{"target": "world"}},
		{logger.DebugLevel, "world", logger.Fields{"say": "hello"}},
	}); err != nil {
		t.Error(err)
	}

	if err := l.Test([]logtest.Entry{{logger.DebugLevel, "world", logger.Fields{"say": "hello"}}}); err != nil {
		t.Error(err)
	}

	expect := strings.Join([]string{
		"unexpected log entries:",
		"expected:",
		"\t[debug]world[say:hello]",
		"but got:",
		"\t[debug]hello[target:world]",
		"\t[debug]world[say:hello]",
		"",
	}, "\n")

	if err := l.TestAll([]logtest.Entry{{logger.DebugLevel, "world", logger.Fields{"say": "hello"}}}); err == nil {
		t.Errorf("expected error but got nil")
	} else if err.Error() != expect {
		t.Errorf("unexpected error:\nexpected:\n```\n%s```\n\nbut got:\n```\n%s```", expect, err.Error())
	}
}

func TestLogTest(t *testing.T) {
	t.Parallel()

	orig := logger.GetLogger()

	lt := logtest.Start()

	if logger.GetLogger() == orig || logger.GetLogger() != lt.Logger {
		t.Fatal("failed to replace logger")
	}

	logger.Debug("hello world", nil)

	if err := lt.Test([]logtest.Entry{{logger.DebugLevel, "hello world", nil}}); err != nil {
		t.Error(err)
	}

	if err := lt.TestAll([]logtest.Entry{{logger.DebugLevel, "hello world", nil}}); err != nil {
		t.Error(err)
	}

	if err := lt.Close(); err != nil {
		t.Fatalf("failed to restore logger: %s", err)
	} else if logger.GetLogger() != orig || logger.GetLogger() == lt.Logger {
		t.Fatal("failed to restore logger")
	}
}
