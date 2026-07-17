package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"strconv"
	"strings"
	"time"
)

// intFmtBufLen is enough for any base-10 int64/uint64: min int64
// "-9223372036854775808" and max uint64 "18446744073709551615" are both 20 bytes.
const intFmtBufLen = 20

// floatFmtBufLen is the largest output strconv.AppendFloat can produce for a
// float64 in 'f' format with 6 decimals: sign + up to 309 integer digits
// (math.MaxFloat64 ≈ 1.8e308) + '.' + 6 fractional digits. Sizing the scratch
// buffer to this keeps WriteFloat allocation-free for the full float64 range.
const floatFmtBufLen = 1 + 309 + 1 + 6

// formatLineInt, formatLineUint, and formatLineFloat build a full protocol line
// from a prefix and numeric value. They are the concatenating fallback used by
// non-buffered writers, which need a single contiguous line anyway; buffered
// writers override the typed methods to append without allocating.
func formatLineInt(prefix string, value int64) string {
	return prefix + strconv.FormatInt(value, 10)
}

func formatLineUint(prefix string, value uint64) string {
	return prefix + strconv.FormatUint(value, 10)
}

func formatLineFloat(prefix string, value float64) string {
	return prefix + strconv.FormatFloat(value, 'f', 6, 64)
}

// Writer that accepts SpectatorD line protocol.
type Writer interface {
	// Write is the primary interface, for meters
	Write(line string)
	// WriteLine writes a protocol line composed of a precomputed prefix
	// (e.g. "c:name,tag=val:") followed by a value (e.g. "1"), without the
	// caller building an intermediate concatenated string. Buffered writers
	// append the two pieces directly into their backing storage, avoiding a
	// per-emit allocation on the metric hot path. Non-buffered writers may
	// concatenate, which is no more expensive than the previous Write path.
	WriteLine(prefix, value string)
	// WriteInt, WriteUint, and WriteFloat write prefix followed by a numeric
	// value formatted with strconv. Buffered writers format directly into their
	// backing storage (strconv.Append*), avoiding the string allocation that a
	// caller-side strconv.Format* + WriteLine would incur. Floats use the same
	// format as the line protocol: 'f', 6 decimal places, 64-bit. Non-buffered
	// writers may format-and-concatenate as a fallback.
	WriteInt(prefix string, value int64)
	WriteUint(prefix string, value uint64)
	WriteFloat(prefix string, value float64)
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
