package writer

import (
	"net"
	"strings"
	"time"

	"github.com/Netflix/spectator-go/v2/spectator/logger"
)

type UnixgramWriter struct {
	raw              *rawUnixgramWriter
	lineBuffer       *LineBuffer
	lowLatencyBuffer *LowLatencyBuffer
}

type rawUnixgramWriter struct {
	addr   *net.UnixAddr
	conn   *net.UnixConn
	logger logger.Logger
}

func (u *rawUnixgramWriter) Write(line string) {
	u.WriteString(line)
}

func (u *rawUnixgramWriter) WriteBytes(line []byte) {
	if u.conn != nil {
		if _, err := u.conn.Write(line); err != nil {
			u.maybeCloseSocket(err)
		}
	} else {
		u.redialSocket()
	}
}

func (u *rawUnixgramWriter) WriteString(line string) {
	if u.conn != nil {
		if _, err := u.conn.Write([]byte(line)); err != nil {
			u.maybeCloseSocket(err)
		}
	} else {
		u.redialSocket()
	}
}

func (u *rawUnixgramWriter) Close() error {
	if u.conn != nil {
		return u.conn.Close()
	}
	return nil
}

func NewUnixgramWriter(path string, logger logger.Logger) (*UnixgramWriter, error) {
	return NewUnixgramWriterWithBuffer(path, logger, 0, 5*time.Second)
}

func NewUnixgramWriterWithBuffer(path string, logger logger.Logger, bufferSize int, flushInterval time.Duration) (*UnixgramWriter, error) {
	addr := &net.UnixAddr{Name: path, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		logger.Errorf("failed to dial unix socket: %v", err)
		conn = nil
	}

	raw := &rawUnixgramWriter{
		addr:   addr,
		conn:   conn,
		logger: logger,
	}
	baseWriter := &UnixgramWriter{
		raw: raw,
	}

	var lineBuffer *LineBuffer
	var lowLatencyBuffer *LowLatencyBuffer
	if bufferSize > 0 && bufferSize <= 65536 {
		// Buffers flush through the raw socket writer, not UnixgramWriter, to avoid
		// re-entering buffering logic on WriteBytes/WriteString.
		lineBuffer = NewLineBuffer(raw, logger, bufferSize, flushInterval)
	} else if bufferSize > 0 {
		// Buffers flush through the raw socket writer, not UnixgramWriter, to avoid
		// re-entering buffering logic on WriteBytes/WriteString.
		lowLatencyBuffer = NewLowLatencyBuffer(raw, logger, bufferSize, flushInterval)
	}
	baseWriter.lineBuffer = lineBuffer
	baseWriter.lowLatencyBuffer = lowLatencyBuffer

	return baseWriter, nil
}

func (u *UnixgramWriter) Write(line string) {
	u.raw.logger.Debugf("Sending line: %s", line)

	if u.lineBuffer != nil {
		u.lineBuffer.Write(line)
		return
	}

	if u.lowLatencyBuffer != nil {
		u.lowLatencyBuffer.Write(line)
		return
	}

	u.raw.WriteString(line)
}

func (u *UnixgramWriter) WriteBytes(line []byte) {
	if u.lineBuffer != nil {
		u.lineBuffer.WriteBytes(line)
		return
	}

	if u.lowLatencyBuffer != nil {
		u.lowLatencyBuffer.WriteBytes(line)
		return
	}

	u.raw.WriteBytes(line)
}

func (u *UnixgramWriter) WriteString(line string) {
	u.raw.WriteString(line)
}

// If anything disturbs access to the unix socket, such as a spectatord process restart (or another
// unknown condition), then all future writes to the unix socket will fail with a "transport endpoint
// is not connected" error.
//
// This means that the UdpWriter is generally more resilient across more operating conditions than the
// UnixgramWriter. The UdpWriter does not continue to fail once it encounters a single failure to write,
// it resumes writing when the port is available again, and it does not require any special connection
// handling.
//
// The addition of reconnect logic to the UnixgramWriter mitigates ongoing issues with unix socket write
// errors. Some packet delivery failure will occur until it can reconnect. With the reconnect logic in
// place, the initialization is now more resilient if the unix socket is not available at program start.
func (u *rawUnixgramWriter) maybeCloseSocket(err error) {
	u.logger.Errorf("failed to write to unix socket: %v\n", err)

	if strings.Contains(err.Error(), "transport endpoint is not connected") {
		u.logger.Infof("close unix socket")
		err := u.conn.Close()
		if err != nil {
			u.logger.Errorf("failed to close unix socket: %v\n", err)
		}
		u.conn = nil
	}
}

func (u *rawUnixgramWriter) redialSocket() {
	u.logger.Infof("re-dial unix socket")

	conn, err := net.DialUnix("unixgram", nil, u.addr)
	if err != nil {
		u.logger.Errorf("failed to dial unix socket: %v", err)
	} else {
		u.conn = conn
	}
}

func (u *UnixgramWriter) Close() error {
	// Stop flush timer, and flush remaining lines
	if u.lineBuffer != nil {
		u.lineBuffer.Close()
	}

	// Stop flush goroutines
	if u.lowLatencyBuffer != nil {
		u.lowLatencyBuffer.Close()
	}

	// Close the connection
	return u.raw.Close()
}
