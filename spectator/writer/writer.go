package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"os"
	"strings"
)

// Writer that accepts SpectatorD line protocol.
type Writer interface {
	Write(line string)
	Close() error
}

type MemoryWriter struct {
	Lines []string
}

func (m *MemoryWriter) Write(line string) {
	m.Lines = append(m.Lines, line)
}

func (m *MemoryWriter) Close() error {
	return nil
}

// NoopWriter is a writer that does nothing.
type NoopWriter struct{}

func (n *NoopWriter) Write(_ string) {}

func (n *NoopWriter) Close() error {
	return nil
}

// StdoutWriter is a writer that writes to stdout.
type StdoutWriter struct{}

func (s *StdoutWriter) Write(line string) {
	_, _ = fmt.Fprintln(os.Stdout, line)
}

func (s *StdoutWriter) Close() error {
	return nil
}

// StderrWriter is a writer that writes to stderr.
type StderrWriter struct{}

func (s *StderrWriter) Write(line string) {
	_, _ = fmt.Fprintln(os.Stderr, line)
}

func (s *StderrWriter) Close() error {
	return nil
}

func IsValidOutputLocation(output string) bool {
	return output == "none" ||
		output == "memory" ||
		output == "stdout" ||
		output == "stderr" ||
		strings.HasPrefix(output, "file://") ||
		strings.HasPrefix(output, "udp://")
}

// NewWriter Create a new writer based on the GetLocation string provided
func NewWriter(outputLocation string, logger logger.Logger) (Writer, error) {
	switch {
	case outputLocation == "none":
		return &NoopWriter{}, nil
	case outputLocation == "memory":
		return &MemoryWriter{}, nil
	case outputLocation == "stdout":
		return &StdoutWriter{}, nil
	case outputLocation == "stderr":
		return &StderrWriter{}, nil
	case strings.HasPrefix(outputLocation, "file://"):
		filePath := strings.TrimPrefix(outputLocation, "file://")
		return NewFileWriter(filePath, logger)
	case strings.HasPrefix(outputLocation, "udp://"):
		address := strings.TrimPrefix(outputLocation, "udp://")
		return NewUdpWriter(address, logger)
	case strings.HasPrefix(outputLocation, "unix://"):
		path := strings.TrimPrefix(outputLocation, "unix://")
		return NewUnixgramWriter(path, logger)
	default:
		return nil, fmt.Errorf("unknown output location: %s", outputLocation)
	}
}
