package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"strings"
	"time"
)

// Writer that accepts SpectatorD line protocol.
type Writer interface {
	// Write is the primary interface, for meters
	Write(line string)
	// WriteBytes and WriteString are secondary interfaces, for buffers
	WriteBytes(line []byte)
	WriteString(line string)
	Close() error
}

func IsValidOutputLocation(output string) bool {
	return output == "none" ||
		output == "memory" ||
		output == "stdout" ||
		output == "stderr" ||
		output == "udp" ||
		output == "unix" ||
		strings.HasPrefix(output, "file://") ||
		strings.HasPrefix(output, "udp://") ||
		strings.HasPrefix(output, "unix://")
}

// NewWriter Create a new writer based on the GetLocation string provided
func NewWriter(outputLocation string, logger logger.Logger) (Writer, error) {
	return NewWriterWithBuffer(outputLocation, logger, 0, 5*time.Second)
}

// NewWriterWithBuffer Create a new writer with buffer support
func NewWriterWithBuffer(outputLocation string, logger logger.Logger, bufferSize int, flushInterval time.Duration) (Writer, error) {
	switch {
	case outputLocation == "none":
		logger.Infof("Initialize NoopWriter")
		return &NoopWriter{}, nil
	case outputLocation == "memory":
		logger.Infof("Initialize MemoryWriter")
		return &MemoryWriter{}, nil
	case outputLocation == "stdout":
		logger.Infof("Initialize StdoutWriter")
		return &StdoutWriter{}, nil
	case outputLocation == "stderr":
		logger.Infof("Initialize StderrWriter")
		return &StderrWriter{}, nil
	case outputLocation == "udp":
		// default udp port for spectatord
		outputLocation = "udp://127.0.0.1:1234"
		logger.Infof("Initialize UdpWriter with address %s", outputLocation)
		address := strings.TrimPrefix(outputLocation, "udp://")
		return NewUdpWriterWithBuffer(address, logger, bufferSize, flushInterval)
	case outputLocation == "unix":
		// default unix domain socket for spectatord
		outputLocation = "unix:///run/spectatord/spectatord.unix"
		logger.Infof("Initialize UnixgramWriter with path %s", outputLocation)
		path := strings.TrimPrefix(outputLocation, "unix://")
		return NewUnixgramWriterWithBuffer(path, logger, bufferSize, flushInterval)
	case strings.HasPrefix(outputLocation, "file://"):
		logger.Infof("Initialize FileWriter with path %s", outputLocation)
		filePath := strings.TrimPrefix(outputLocation, "file://")
		return NewFileWriter(filePath, logger)
	case strings.HasPrefix(outputLocation, "udp://"):
		logger.Infof("Initialize UdpWriter with address %s", outputLocation)
		address := strings.TrimPrefix(outputLocation, "udp://")
		return NewUdpWriterWithBuffer(address, logger, bufferSize, flushInterval)
	case strings.HasPrefix(outputLocation, "unix://"):
		logger.Infof("Initialize UnixgramWriter with path %s", outputLocation)
		path := strings.TrimPrefix(outputLocation, "unix://")
		return NewUnixgramWriterWithBuffer(path, logger, bufferSize, flushInterval)
	default:
		return nil, fmt.Errorf("unknown output location: %s", outputLocation)
	}
}
