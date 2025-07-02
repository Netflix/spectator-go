package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"net"
	"strings"
)

type UnixgramWriter struct {
	addr       *net.UnixAddr
	conn       *net.UnixConn
	logger     logger.Logger
	lineBuffer *LineBuffer
}

type unixgramDirectWriter struct {
	*UnixgramWriter
}

func (u *unixgramDirectWriter) Write(line string) {
	u.UnixgramWriter.writeDirectly(line)
}

func (u *unixgramDirectWriter) Close() error {
	if u.UnixgramWriter.conn != nil {
		return u.UnixgramWriter.conn.Close()
	}
	return nil
}

func NewUnixgramWriter(path string, logger logger.Logger) (*UnixgramWriter, error) {
	return NewUnixgramWriterWithBuffer(path, logger, 0)
}

func NewUnixgramWriterWithBuffer(path string, logger logger.Logger, bufferSize int) (*UnixgramWriter, error) {
	addr := &net.UnixAddr{Name: path, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		logger.Errorf("failed to dial unix socket: %v", err)
		conn = nil
	}

	baseWriter := &UnixgramWriter{
		addr:   addr,
		conn:   conn,
		logger: logger,
	}

	var lineBuffer *LineBuffer
	if bufferSize > 0 {
		lineBuffer = NewLineBuffer(&unixgramDirectWriter{baseWriter}, bufferSize, logger)
	}

	baseWriter.lineBuffer = lineBuffer
	return baseWriter, nil
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
func (u *UnixgramWriter) Write(line string) {
	if u.lineBuffer != nil {
		u.lineBuffer.Write(line)
		return
	}

	u.writeDirectly(line)
}

func (u *UnixgramWriter) writeDirectly(line string) {
	u.logger.Debugf("Sending line: %s", line)

	if u.conn != nil {
		if _, err := u.conn.Write([]byte(line)); err != nil {
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
	} else {
		u.logger.Infof("re-dial unix socket")

		conn, err := net.DialUnix("unixgram", nil, u.addr)
		if err != nil {
			u.logger.Errorf("failed to dial unix socket: %v", err)
		} else {
			u.conn = conn
		}
	}
}

func (u *UnixgramWriter) Close() error {
	if u.lineBuffer != nil {
		if err := u.lineBuffer.Close(); err != nil {
			return err
		}
	}
	if u.conn != nil {
		return u.conn.Close()
	}
	return nil
}
