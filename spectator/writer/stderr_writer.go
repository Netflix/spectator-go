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

func (s *StderrWriter) WriteBytes(line []byte) {
	s.WriteString(string(line))
}

func (s *StderrWriter) WriteString(line string) {
	_, _ = fmt.Fprintln(os.Stderr, line)
}

func (s *StderrWriter) Close() error {
	return nil
}
