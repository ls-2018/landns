package logger_test

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns/logger"
)

func TestLevel(t *testing.T) {
	tests := []struct {
		Level  logger.Level
		Expect string
	}{
		{logger.DebugLevel, "debug"},
		{logger.InfoLevel, "info"},
		{logger.WarnLevel, "warning"},
		{logger.ErrorLevel, "error"},
		{logger.FatalLevel, "fatal"},
	}

	for _, tt := range tests {
		if tt.Level.String() != tt.Expect {
			t.Errorf("failed to convert to human readable string: expected %#v but got %#v", tt.Expect, tt.Level.String())
		}
	}
}

func TestLevel_UnmarshalText(t *testing.T) {
	tests := []struct {
		Str string
		Expect logger.Level
	}{
		{"debug", logger.DebugLevel},
		{"DEBUG", logger.DebugLevel},
		{"info", logger.InfoLevel},
		{"INFORMATION", logger.InfoLevel},
		{"Warn", logger.WarnLevel},
		{"warning", logger.WarnLevel},
		{"error", logger.ErrorLevel},
		{"FATAL", logger.FatalLevel},
	}

	for _, tt := range tests {
		var ll logger.Level

		if err := (&ll).UnmarshalText([]byte(tt.Str)); err != nil {
			t.Errorf("failed to unmarshal text %#v: %s", tt.Str, err)
		} else if ll != tt.Expect {
			t.Errorf("unexpect unmarshaled result: expected %s but got %s", tt.Expect, ll)
		}
	}

	var ll logger.Level
	if err := (&ll).UnmarshalText([]byte("no such level")); err == nil {
		t.Errorf("expected error but not reported")
	}
}

type LoggerTestEntry struct {
	Level string
	Func  func(string, logger.Fields)
	Exit  bool
}

type LoggerTest []LoggerTestEntry

func (tests LoggerTest) Run(t *testing.T, buf *bytes.Buffer, l *logger.BasicLogger) {
	exit := false
	l.Logger.ExitFunc = func(int) {
		exit = true
	}

	for _, tt := range tests {
		buf.Reset()
		exit = false
		tt.Func("hello", logger.Fields{
			"target": "world",
			"id":     1,
		})

		expect := fmt.Sprintf(`time="20..-..-..T..:..:..([+-]..:..|Z)" level=%s msg=hello id=1 target=world\n`, tt.Level)

		if ok, err := regexp.MatchString("^"+expect+"$", buf.String()); err != nil {
			t.Errorf("failed to compare: %s", err)
		} else if !ok {
			t.Errorf("failed to logging %s:\nexpected: %#v\nbut got:  %#v", tt.Level, expect, buf.String())
		}

		if exit != tt.Exit {
			t.Errorf("unexpected exit status: expected %v but got %v", tt.Exit, exit)
		}
	}
}

func TestBasicLogger(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	l := logger.New(buf, logger.DebugLevel)

	LoggerTest{
		{"debug", l.Debug, false},
		{"info", l.Info, false},
		{"warning", l.Warn, false},
		{"error", l.Error, false},
		{"fatal", l.Fatal, true},
	}.Run(t, buf, l)
}

func TestBasicLogger_level(t *testing.T) {
	levels := []logger.Level{
		logger.FatalLevel,
		logger.ErrorLevel,
		logger.WarnLevel,
		logger.InfoLevel,
		logger.DebugLevel,
	}

	for i, level := range levels {
		buf := bytes.NewBuffer([]byte{})
		l := logger.New(buf, level)
		l.Logger.ExitFunc = func(int) {}

		if l.GetLevel() != level {
			t.Errorf("failed to get level: expected %s but got %s", level, l.GetLevel())
		}

		l.Debug("d", logger.Fields{})
		l.Info("i", logger.Fields{})
		l.Warn("w", logger.Fields{})
		l.Error("e", logger.Fields{})
		l.Fatal("f", logger.Fields{})

		if strings.Count(buf.String(), "\n") != i+1 {
			t.Errorf("unexpected log length:\nexpected %d lines\nbut got:\n%s", i+1, buf.String())
		}
	}
}

func TestDefaultLogger(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	l := logger.New(buf, logger.DebugLevel)
	logger.SetLogger(l)

	if logger.GetLogger() != l {
		t.Fatalf("failed to set/get default logger")
	}

	LoggerTest{
		{"debug", logger.Debug, false},
		{"info", logger.Info, false},
		{"warning", logger.Warn, false},
		{"error", logger.Error, false},
		{"fatal", logger.Fatal, true},
	}.Run(t, buf, l)
}
