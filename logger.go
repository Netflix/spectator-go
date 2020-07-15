package spectator

import (
	"log"
	"os"
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
	debug *log.Logger
	info  *log.Logger
	error *log.Logger
}

func defaultLogger() *DefaultLogger {
	flags := log.LstdFlags

	debug := log.New(os.Stdout, "DEBUG: ", flags)
	info := log.New(os.Stdout, "INFO: ", flags)
	err := log.New(os.Stdout, "ERROR: ", flags)

	return &DefaultLogger{
		debug,
		info,
		err,
	}
}

// Debugf is for debug level messages. Satisfies Logger interface.
func (l *DefaultLogger) Debugf(format string, v ...interface{}) {
	l.debug.Printf(format, v...)
}

// Infof is for info level messages. Satisfies Logger interface.
func (l *DefaultLogger) Infof(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

// Errorf is for error level messages. Satisfies Logger interface.
func (l *DefaultLogger) Errorf(format string, v ...interface{}) {
	l.error.Printf(format, v...)
}
