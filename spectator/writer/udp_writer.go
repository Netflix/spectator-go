package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"net"
	"time"
)

type UdpWriter struct {
	conn             *net.UDPConn
	logger           logger.Logger
	lineBuffer       *LineBuffer
	lowLatencyBuffer *LowLatencyBuffer
}

type udpBufferWriter struct {
	*UdpWriter
}

func NewUdpWriter(address string, logger logger.Logger) (*UdpWriter, error) {
	return NewUdpWriterWithBuffer(address, logger, 0, 5*time.Second)
}

func NewUdpWriterWithBuffer(address string, logger logger.Logger, bufferSize int, flushInterval time.Duration) (*UdpWriter, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	baseWriter := &UdpWriter{
		conn:   conn,
		logger: logger,
	}

	var lineBuffer *LineBuffer
	var lowLatencyBuffer *LowLatencyBuffer
	if bufferSize > 0 && bufferSize <= 65536 {
		lineBuffer = NewLineBuffer(&udpBufferWriter{baseWriter}, logger, bufferSize, flushInterval)
	} else if bufferSize > 0 {
		lowLatencyBuffer = NewLowLatencyBuffer(&udpBufferWriter{baseWriter}, logger, bufferSize, flushInterval)
	}
	baseWriter.lineBuffer = lineBuffer
	baseWriter.lowLatencyBuffer = lowLatencyBuffer

	return baseWriter, nil
}

func (u *UdpWriter) Write(line string) {
	u.logger.Debugf("Sending line: %s", line)

	if u.lineBuffer != nil {
		u.lineBuffer.Write(line)
		return
	}

	if u.lowLatencyBuffer != nil {
		u.lowLatencyBuffer.Write(line)
		return
	}

	u.WriteString(line)
}

func (u *UdpWriter) WriteBytes(line []byte) {
	_, err := u.conn.Write(line)
	if err != nil {
		u.logger.Errorf("Error writing to UDP: %s", err)
	}
}

func (u *UdpWriter) WriteString(line string) {
	_, err := u.conn.Write([]byte(line))
	if err != nil {
		u.logger.Errorf("Error writing to UDP: %s", err)
	}
}

func (u *UdpWriter) Close() error {
	// Stop flush timer, and flush remaining lines
	if u.lineBuffer != nil {
		u.lineBuffer.Close()
	}

	// Stop flush goroutines
	if u.lowLatencyBuffer != nil {
		u.lowLatencyBuffer.Close()
	}

	// Close the connection, if it exists
	if u.conn != nil {
		return u.conn.Close()
	}

	return nil
}
