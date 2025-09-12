package logger

import (
	"fmt"
	"log/slog"
	"sync/atomic"
)

var zmsLogger atomic.Pointer[ZMSLogger]

func init() {
	zmsLogger.Store(NewZMSLogger())
}

type ZMSLogger struct {
	slogger *slog.Logger
}

func NewZMSLogger() *ZMSLogger {
	return &ZMSLogger{
		slogger: slog.Default(),
	}
	// return &ZMSLogger{}
}

func Default() *ZMSLogger {
	return zmsLogger.Load()
}

func SetLogLevel(level slog.Level) {
	slog.SetLogLoggerLevel(level)
}

// slog wrapper

func Debug(msg string, args ...any) {
	zmsLogger.Load().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	zmsLogger.Load().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	zmsLogger.Load().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	zmsLogger.Load().Error(msg, args...)
}

func (l *ZMSLogger) Debug(msg string, args ...any) {
	l.slogger.Debug(msg, args...)
}

func (l *ZMSLogger) Info(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

func (l *ZMSLogger) Warn(msg string, args ...any) {
	l.slogger.Warn(msg, args...)
}

func (l *ZMSLogger) Error(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

// badger.logger

func (l *ZMSLogger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.slogger.Error(msg)
}

func (l *ZMSLogger) Warningf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.slogger.Warn(msg)
}

func (l *ZMSLogger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.slogger.Info(msg)
}

func (l *ZMSLogger) Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.slogger.Debug(msg)
}

// tailf.logger

func (l *ZMSLogger) Fatal(v ...interface{}) {
	pairs := genericPairs(v...)
	l.slogger.Error("An error occured", pairs...)
}

func (l *ZMSLogger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.slogger.Error(msg)
}
func (l *ZMSLogger) Fatalln(v ...interface{}) {
	l.slogger.Error("An error occured", v...)
}
func (l *ZMSLogger) Panic(v ...interface{}) {
	pairs := genericPairs(v...)
	l.slogger.Error("", pairs...)
}
func (l *ZMSLogger) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.slogger.Error(msg)
}
func (l *ZMSLogger) Panicln(v ...interface{}) {
	l.slogger.Error("An error occured", v...)
}
func (l *ZMSLogger) Print(v ...interface{}) {
	pairs := genericPairs(v...)
	l.slogger.Info("", pairs...)
}
func (l *ZMSLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.slogger.Info(msg)
}
func (l *ZMSLogger) Println(v ...interface{}) {
	l.slogger.Info("", v...)
}

func genericPairs(v ...interface{}) []any {
	pairs := make([]any, 0, len(v)/2)
	for i := 0; i < len(v)-1; i += 2 {
		key, ok := v[i].(string)
		if !ok {
			key = fmt.Sprintf("non_string_key_%d", i)
		}
		pairs = append(pairs, slog.Any(key, v[i+1]))
	}
	return pairs
}
