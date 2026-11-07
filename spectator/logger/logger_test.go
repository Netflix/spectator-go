package logger

import (
	"bytes"
	"log/slog"
	"testing"
)

type panicOnFormat struct{}

func (panicOnFormat) String() string {
	panic("formatted a disabled log message")
}

func withHandler(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}

func TestDefaultLoggerEnabledLevels(t *testing.T) {
	buf := withHandler(t, slog.LevelDebug)
	l := NewDefaultLogger()
	l.Debugf("debug %d", 1)
	l.Infof("info %s", "two")
	l.Errorf("error %v", 3)

	out := buf.String()
	for _, want := range []string{"msg=\"debug 1\"", "msg=\"info two\"", "msg=\"error 3\""} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("expected %q in output: %s", want, out)
		}
	}
}

func TestDefaultLoggerDisabledLevelsDoNotFormat(t *testing.T) {
	buf := withHandler(t, slog.LevelError+1)
	l := NewDefaultLogger()
	l.Debugf("%v", panicOnFormat{})
	l.Infof("%v", panicOnFormat{})
	l.Errorf("%v", panicOnFormat{})

	if buf.Len() != 0 {
		t.Errorf("expected no output, got: %s", buf.String())
	}
}
