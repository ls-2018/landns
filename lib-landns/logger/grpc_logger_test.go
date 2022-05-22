package logger_test

import (
	"bytes"
	"testing"

	"github.com/macrat/landns/lib-landns/logger"
	"github.com/macrat/landns/lib-landns/logger/logtest"
)

func TestGRPCLogger(t *testing.T) {
	type A func(...interface{})
	type B func(string, ...interface{})

	tests := []struct {
		Level   logger.Level
		Writers func(logger.GRPCLogger) (A, A, B)
	}{
		{logger.InfoLevel, func(l logger.GRPCLogger) (A, A, B) {
			return l.Info, l.Infoln, l.Infof
		}},
		{logger.WarnLevel, func(l logger.GRPCLogger) (A, A, B) {
			return l.Warning, l.Warningln, l.Warningf
		}},
		{logger.ErrorLevel, func(l logger.GRPCLogger) (A, A, B) {
			return l.Error, l.Errorln, l.Errorf
		}},
		{logger.FatalLevel, func(l logger.GRPCLogger) (A, A, B) {
			return l.Fatal, l.Fatalln, l.Fatalf
		}},
	}

	for _, tt := range tests {
		fields := logger.Fields{"id": 1}
		dl := new(logtest.DummyLogger)
		l := logger.GRPCLogger{Logger: dl, Fields: fields}

		Print, Println, Printf := tt.Writers(l)
		Print("hello %s", "world")
		Println("hello %s", "world")
		Printf("hello %s", "world")

		err := dl.TestAll([]logtest.Entry{
			{tt.Level, "hello %sworld", fields},
			{tt.Level, "hello %sworld", fields},
			{tt.Level, "hello world", fields},
		})
		if err != nil {
			t.Error(err.Error())
		}
	}
}

func TestGRPCLogger_getLogger(t *testing.T) {
	defaultLogger := new(logtest.DummyLogger)
	fieldLogger := new(logtest.DummyLogger)

	logger.SetLogger(defaultLogger)

	logger.GRPCLogger{Logger: fieldLogger}.Info("hello world")
	fieldLogger.TestAll([]logtest.Entry{
		{logger.InfoLevel, "hello world", nil},
	})

	logger.GRPCLogger{}.Info("hello world")
	fieldLogger.TestAll([]logtest.Entry{
		{logger.InfoLevel, "hello world", nil},
	})
}

const (
	grpcInfoLevel int = iota
	grpcWarningLevel
	grpcErrorLevel
	grpcFatalLevel
)

var (
	grpcLevels = []int{grpcInfoLevel, grpcWarningLevel, grpcErrorLevel, grpcFatalLevel, 999}
)

func TestGRPCLogger_V(t *testing.T) {
	tests := []struct {
		Level  logger.Level
		Expect []bool
	}{
		{logger.DebugLevel, []bool{true, false, false, false, false}},
		{logger.InfoLevel, []bool{true, false, false, false, false}},
		{logger.WarnLevel, []bool{true, true, false, false, false}},
		{logger.ErrorLevel, []bool{true, true, true, false, false}},
		{logger.FatalLevel, []bool{true, true, true, true, false}},
	}

	buf := bytes.NewBuffer([]byte{})
	for _, tt := range tests {
		l := logger.GRPCLogger{Logger: logger.New(buf, tt.Level)}

		for i := range grpcLevels {
			v := l.V(grpcLevels[i])
			if v != tt.Expect[i] {
				t.Errorf("unexpected V result: expected %v but got %v", tt.Expect[i], v)
			}
		}
	}
}
