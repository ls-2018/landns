package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Level is logging level like Fatal or Warn.
type Level uint

// String is converter to human readable string.
func (ll Level) String() string {
	return logrus.Level(ll).String()
}

// UnmarshalText is convert from bytes string.
func (ll *Level) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "debug":
		*ll = DebugLevel
	case "info", "information":
		*ll = InfoLevel
	case "warn", "warning":
		*ll = WarnLevel
	case "error":
		*ll = ErrorLevel
	case "fatal":
		*ll = FatalLevel
	default:
		return fmt.Errorf("unknown level: %s", string(text))
	}
	return nil
}

const (
	// FatalLevel is log level for critical error.
	FatalLevel = Level(logrus.FatalLevel)

	// ErrorLevel is log level for normally error.
	ErrorLevel = Level(logrus.ErrorLevel)

	// WarnLevel is log level for need a little of attention error.
	WarnLevel = Level(logrus.WarnLevel)

	// InfoLevel is log level for some useful information log.
	InfoLevel = Level(logrus.InfoLevel)

	// DebugLevel is log level for informations for debugging.
	DebugLevel = Level(logrus.DebugLevel)
)

var logger Logger = New(os.Stderr, WarnLevel)

// SetLogger is the default logger setter.
func SetLogger(l Logger) {
	logger = l
}

// GetLogger is the default logger getter.
func GetLogger() Logger {
	return logger
}

// Debug is default writer to DebugLevel log.
func Debug(message string, fields Fields) {
	logger.Debug(message, fields)
}

// Info is default writer to InfoLevel log.
func Info(message string, fields Fields) {
	logger.Info(message, fields)
}

// Warn is default writer to WarnLevel log.
func Warn(message string, fields Fields) {
	logger.Warn(message, fields)
}

// Error is default writer to ErrorLevel log.
func Error(message string, fields Fields) {
	logger.Error(message, fields)
}

// Fatal is default writer to FatalLevel log.
func Fatal(message string, fields Fields) {
	logger.Fatal(message, fields)
}

// Fields is annotations for a log entry.
type Fields map[string]interface{}

// Logger is the interface of log writer.
type Logger interface {
	Debug(string, Fields)
	Info(string, Fields)
	Warn(string, Fields)
	Error(string, Fields)
	Fatal(string, Fields)
	GetLevel() Level
}

// BasicLogger is the default implements of Logger.
//
// It's using github.com/sirupsen/logrus.
type BasicLogger struct {
	Logger *logrus.Logger
}

// New is constructor of BasicLogger.
func New(out io.Writer, level Level) *BasicLogger {
	l := logrus.New()
	l.SetOutput(out)
	l.SetLevel(logrus.Level(level))

	return &BasicLogger{l}
}

// Debug is writer to DebugLevel log.
func (l *BasicLogger) Debug(message string, fields Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Debug(message)
}

// Info is writer to InfoLevel log.
func (l *BasicLogger) Info(message string, fields Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Info(message)
}

// Warn is writer to WarnLevel log.
func (l *BasicLogger) Warn(message string, fields Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Warn(message)
}

// Error is writer to ErrorLevel log.
func (l *BasicLogger) Error(message string, fields Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Error(message)
}

// Fatal is writer to FatalLevel log.
func (l *BasicLogger) Fatal(message string, fields Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Fatal(message)
}

// GetLevel is getter to current log level.
func (l *BasicLogger) GetLevel() Level {
	return Level(l.Logger.GetLevel())
}
