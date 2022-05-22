package logger

import (
	"fmt"
)

// GRPCLogger is implements of LoggerV2 of google.golang.org/grpc/grpclog
type GRPCLogger struct {
	Logger Logger
	Fields Fields
}

func (l GRPCLogger) getLogger() Logger {
	if l.Logger != nil {
		return l.Logger
	}
	return logger
}

// Info is logging as InfoLevel with l.Fields. Arguments are handled in fmt.Sprint.
func (l GRPCLogger) Info(args ...interface{}) {
	l.getLogger().Info(fmt.Sprint(args...), l.Fields)
}

// Infoln is same as Info.
func (l GRPCLogger) Infoln(args ...interface{}) {
	l.Info(args...)
}

// Infof is logging as InfoLevel with l.Fields. Arguments are handled in fmt.Sprintf.
func (l GRPCLogger) Infof(format string, args ...interface{}) {
	l.getLogger().Info(fmt.Sprintf(format, args...), l.Fields)
}

// Warning is logging as WarnLevel with l.Fields. Arguments are handled in fmt.Sprint.
func (l GRPCLogger) Warning(args ...interface{}) {
	l.getLogger().Warn(fmt.Sprint(args...), l.Fields)
}

// Warningln is same as Warning.
func (l GRPCLogger) Warningln(args ...interface{}) {
	l.Warning(args...)
}

// Warningf is logging as WarnLevel with l.Fields. Arguments are handled in fmt.Sprintf.
func (l GRPCLogger) Warningf(format string, args ...interface{}) {
	l.getLogger().Warn(fmt.Sprintf(format, args...), l.Fields)
}

// Error is logging as ErrorLevel with l.Fields. Arguments are handled in fmt.Sprint.
func (l GRPCLogger) Error(args ...interface{}) {
	l.getLogger().Error(fmt.Sprint(args...), l.Fields)
}

// Errorln is same as Error.
func (l GRPCLogger) Errorln(args ...interface{}) {
	l.Error(args...)
}

// Errorf is logging as ErrorLevel with l.Fields. Arguments are handled in fmt.Sprintf.
func (l GRPCLogger) Errorf(format string, args ...interface{}) {
	l.getLogger().Error(fmt.Sprintf(format, args...), l.Fields)
}

// Fatal is logging as FatalLevel with l.Fields. Arguments are handled in fmt.Sprint.
func (l GRPCLogger) Fatal(args ...interface{}) {
	l.getLogger().Fatal(fmt.Sprint(args...), l.Fields)
}

// Fatalln is same as Fatal.
func (l GRPCLogger) Fatalln(args ...interface{}) {
	l.Fatal(args...)
}

// Fatalf is logging as FatalLevel with l.Fields. Arguments are handled in fmt.Sprintf.
func (l GRPCLogger) Fatalf(format string, args ...interface{}) {
	l.getLogger().Fatal(fmt.Sprintf(format, args...), l.Fields)
}

// V is reports whether level will print or not.
func (l GRPCLogger) V(lv int) bool {
	switch l.getLogger().GetLevel() {
	case DebugLevel, InfoLevel:
		return lv <= 0
	case WarnLevel:
		return lv <= 1
	case ErrorLevel:
		return lv <= 2
	default:
		return lv <= 3
	}
}
