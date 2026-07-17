package writer

import (
	"fmt"
	"os"
)

// StderrWriter is a writer that writes to stderr.
type StderrWriter struct{}

func (s *StderrWriter) Write(line string) {
	s.WriteString(line)
}

func (s *StderrWriter) WriteLine(prefix, value string) {
	s.WriteString(prefix + value)
}

func (s *StderrWriter) WriteInt(prefix string, value int64) {
	s.WriteString(formatLineInt(prefix, value))
}

func (s *StderrWriter) WriteUint(prefix string, value uint64) {
	s.WriteString(formatLineUint(prefix, value))
}

func (s *StderrWriter) WriteFloat(prefix string, value float64) {
	s.WriteString(formatLineFloat(prefix, value))
}

func (s *StderrWriter) WriteBytes(line []byte) {
	s.WriteString(string(line))
}

func (s *StderrWriter) WriteString(line string) {
	_, _ = fmt.Fprintln(os.Stderr, line)
}

func (s *StderrWriter) Close() error {
	return nil
}
