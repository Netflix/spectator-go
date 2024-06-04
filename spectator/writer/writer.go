package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"os"
	"slices"
	"strings"
	"sync"
)

// Writer that accepts SpectatorD line protocol.
type Writer interface {
	Write(line string)
	Close() error
}

type MemoryWriter struct {
	lines []string
	mu    sync.RWMutex
}

func (m *MemoryWriter) Write(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lines = append(m.lines, line)
}

func (m *MemoryWriter) Lines() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.lines)
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
		logger.Infof("Initializing NoopWriter")
		return &NoopWriter{}, nil
	case outputLocation == "memory":
		logger.Infof("Initializing MemoryWriter")
		return &MemoryWriter{}, nil
	case outputLocation == "stdout":
		logger.Infof("Initializing StdoutWriter")
		return &StdoutWriter{}, nil
	case outputLocation == "stderr":
		logger.Infof("Initializing StderrWriter")
		return &StderrWriter{}, nil
	case strings.HasPrefix(outputLocation, "file://"):
		logger.Infof("Initializing FileWriter with path %s", outputLocation)
		filePath := strings.TrimPrefix(outputLocation, "file://")
		return NewFileWriter(filePath, logger)
	case strings.HasPrefix(outputLocation, "udp://"):
		logger.Infof("Initializing UdpWriter with address %s", outputLocation)
		address := strings.TrimPrefix(outputLocation, "udp://")
		return NewUdpWriter(address, logger)
	case strings.HasPrefix(outputLocation, "unix://"):
		logger.Infof("Initializing UnixgramWriter with path %s", outputLocation)
		path := strings.TrimPrefix(outputLocation, "unix://")
		return NewUnixgramWriter(path, logger)
	default:
		return nil, fmt.Errorf("unknown output location: %s", outputLocation)
	}
}
