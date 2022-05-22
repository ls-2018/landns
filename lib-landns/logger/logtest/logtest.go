// Package logtest is the dummy logger for testing Landns.
package logtest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/macrat/landns/lib-landns/logger"
)

// Entry is one entry of DummyLogger.
type Entry struct {
	Level   logger.Level
	Message string
	Fields  logger.Fields
}

// String is converter to human readable string.
func (le Entry) String() string {
	return fmt.Sprintf("[%s]%s%s", le.Level, le.Message, fmt.Sprint(le.Fields)[3:])
}

// DummyLogger is dummy logger for logging test.
type DummyLogger []Entry

// String is converter to human readable string.
func (l *DummyLogger) String() string {
	ss := make([]string, len(*l))
	for i := range *l {
		ss[i] = (*l)[i].String()
	}
	return strings.Join(ss, "\n")
}

// Debug is writer to DebugLevel log.
func (l *DummyLogger) Debug(message string, fields logger.Fields) {
	*l = append(*l, Entry{
		Level:   logger.DebugLevel,
		Message: message,
		Fields:  fields,
	})
}

// Info is writer to InfoLevel log.
func (l *DummyLogger) Info(message string, fields logger.Fields) {
	*l = append(*l, Entry{
		Level:   logger.InfoLevel,
		Message: message,
		Fields:  fields,
	})
}

// Warn is writer to WarnLevel log.
func (l *DummyLogger) Warn(message string, fields logger.Fields) {
	*l = append(*l, Entry{
		Level:   logger.WarnLevel,
		Message: message,
		Fields:  fields,
	})
}

// Error is writer to ErrorLevel log.
func (l *DummyLogger) Error(message string, fields logger.Fields) {
	*l = append(*l, Entry{
		Level:   logger.ErrorLevel,
		Message: message,
		Fields:  fields,
	})
}

// Fatal is writer to FatalLevel log.
func (l *DummyLogger) Fatal(message string, fields logger.Fields) {
	*l = append(*l, Entry{
		Level:   logger.FatalLevel,
		Message: message,
		Fields:  fields,
	})
}

// GetLevel is always returns logger.DebugLevel.
func (l *DummyLogger) GetLevel() logger.Level {
	return logger.DebugLevel
}

func (l *DummyLogger) makeError(expect []Entry) error {
	msg := "unexpected log entries:\nexpected:\n"
	for _, e := range expect {
		msg += "\t" + e.String() + "\n"
	}
	msg += "but got:\n"
	for _, e := range *l {
		msg += "\t" + e.String() + "\n"
	}
	return errors.New(msg)
}

// Test is check last log entries.
func (l *DummyLogger) Test(expect []Entry) error {
	if len(*l) < len(expect) {
		return l.makeError(expect)
	}

	trim := (*l)[len(*l)-len(expect):]
	return trim.TestAll(expect)
}

// TestAll is check all log entries.
func (l *DummyLogger) TestAll(expect []Entry) error {
	if len(*l) != len(expect) {
		return l.makeError(expect)
	}

	for i, e := range *l {
		if expect[i].String() != e.String() {
			return l.makeError(expect)
		}
	}

	return nil
}

// LogTest is the helper to log testing.
type LogTest struct {
	Logger         *DummyLogger
	originalLogger logger.Logger
}

// Start is starter of LogTest.
func Start() LogTest {
	lt := LogTest{new(DummyLogger), logger.GetLogger()}
	logger.SetLogger(lt.Logger)
	return lt
}

// Close is restore logger state to before call Start.
func (lt LogTest) Close() error {
	logger.SetLogger(lt.originalLogger)
	return nil
}

// Test is check last log entries.
func (lt LogTest) Test(expect []Entry) error {
	return lt.Logger.Test(expect)
}

// TestAll is check all log entries.
func (lt LogTest) TestAll(expect []Entry) error {
	return lt.Logger.TestAll(expect)
}
