package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"net"
)

type UdpWriter struct {
	conn   *net.UDPConn
	logger logger.Logger
}

func NewUdpWriter(address string, logger logger.Logger) (*UdpWriter, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	return &UdpWriter{conn, logger}, nil
}

func (u *UdpWriter) Write(line string) {
	u.logger.Debugf("Sending line: %s", line)

	// Methods on conn are thread-safe
	_, err := u.conn.Write([]byte(line))
	if err != nil {
		u.logger.Errorf("Error writing to UDP: %s", err)
	}
}

func (u *UdpWriter) Close() error {
	return u.conn.Close()
}
