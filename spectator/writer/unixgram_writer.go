package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/spectator/logger"
	"net"
)

type UnixgramWriter struct {
	conn   *net.UnixConn
	logger logger.Logger
}

func NewUnixgramWriter(path string, logger logger.Logger) (*UnixgramWriter, error) {
	addr := &net.UnixAddr{Name: path, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial unix socket: %v", err)
	}

	return &UnixgramWriter{conn, logger}, nil
}

func (u *UnixgramWriter) Write(line string) {
	u.logger.Debugf("Sending line: %s", line)

	if _, err := u.conn.Write([]byte(line)); err != nil {
		fmt.Printf("failed to write to unix socket: %v\n", err)
	}
}

func (u *UnixgramWriter) Close() error {
	return u.conn.Close()
}
