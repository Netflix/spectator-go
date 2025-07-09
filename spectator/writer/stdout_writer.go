package writer

import (
	"fmt"
	"os"
)

// StdoutWriter is a writer that writes to stdout.
type StdoutWriter struct{}

func (s *StdoutWriter) Write(line string) {
	s.WriteString(line)
}

func (s *StdoutWriter) WriteBytes(line []byte) {
	s.WriteString(string(line))
}

func (s *StdoutWriter) WriteString(line string) {
	_, _ = fmt.Fprintln(os.Stdout, line)
}

func (s *StdoutWriter) Close() error {
	return nil
}
