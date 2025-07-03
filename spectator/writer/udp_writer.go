package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"net"
)

type UdpWriter struct {
	conn       *net.UDPConn
	logger     logger.Logger
	lineBuffer *LineBuffer
}

type udpDirectWriter struct {
	*UdpWriter
}

func (u *udpDirectWriter) Write(line string) {
	u.UdpWriter.writeDirectly(line)
}

func (u *udpDirectWriter) Close() error {
	if u.UdpWriter.conn != nil {
		return u.UdpWriter.conn.Close()
	}
	return nil
}

func NewUdpWriter(address string, logger logger.Logger) (*UdpWriter, error) {
	return NewUdpWriterWithBuffer(address, logger, 0)
}

func NewUdpWriterWithBuffer(address string, logger logger.Logger, bufferSize int) (*UdpWriter, error) {
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
	if bufferSize > 0 {
		lineBuffer = NewLineBuffer(&udpDirectWriter{baseWriter}, bufferSize, logger)
	}

	baseWriter.lineBuffer = lineBuffer
	return baseWriter, nil
}

func (u *UdpWriter) Write(line string) {
	if u.lineBuffer != nil {
		u.lineBuffer.Write(line)
		return
	}

	u.writeDirectly(line)
}

func (u *UdpWriter) writeDirectly(line string) {
	u.logger.Debugf("Sending line: %s", line)

	// Methods on conn are thread-safe
	_, err := u.conn.Write([]byte(line))
	if err != nil {
		u.logger.Errorf("Error writing to UDP: %s", err)
	}
}

func (u *UdpWriter) Close() error {
	if u.lineBuffer != nil {
		if err := u.lineBuffer.Close(); err != nil {
			return err
		}
	}
	return u.conn.Close()
}
