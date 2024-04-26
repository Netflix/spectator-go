package logger

import (
	"fmt"
	"log/slog"
)

// Logger represents the shape of the logging dependency that the spectator
// library expects.
type Logger interface {
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// DefaultLogger is a plain text stdout logger.
type DefaultLogger struct {
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

// Debugf is for debug level messages. Satisfies Logger interface.
func (l *DefaultLogger) Debugf(format string, v ...interface{}) {
	slog.Debug(fmt.Sprintf(format, v...))
}

// Infof is for info level messages. Satisfies Logger interface.
func (l *DefaultLogger) Infof(format string, v ...interface{}) {
	slog.Info(fmt.Sprintf(format, v...))
}

// Errorf is for error level messages. Satisfies Logger interface.
func (l *DefaultLogger) Errorf(format string, v ...interface{}) {
	slog.Error(fmt.Sprintf(format, v...))
}
