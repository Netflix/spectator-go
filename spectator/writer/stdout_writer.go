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

func (s *StdoutWriter) WriteLine(prefix, value string) {
	s.WriteString(prefix + value)
}

func (s *StdoutWriter) WriteInt(prefix string, value int64) {
	s.WriteString(formatLineInt(prefix, value))
}

func (s *StdoutWriter) WriteUint(prefix string, value uint64) {
	s.WriteString(formatLineUint(prefix, value))
}

func (s *StdoutWriter) WriteFloat(prefix string, value float64) {
	s.WriteString(formatLineFloat(prefix, value))
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
