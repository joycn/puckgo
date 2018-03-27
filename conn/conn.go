package conn

import (
	"fmt"
	"net"
	"time"
)

type Dialer interface {
	// Dial connects to the given address via the proxy.
	Dial(network, addr string) (c net.Conn, err error)
}

type IdleTimeoutConn struct {
	*net.TCPConn
	Timeout time.Duration
}

func (c *IdleTimeoutConn) Read(buf []byte) (int, error) {
	c.SetReadDeadline(time.Now().Add(c.Timeout))
	return c.TCPConn.Read(buf)
}

func (c *IdleTimeoutConn) Write(buf []byte) (int, error) {
	c.SetWriteDeadline(time.Now().Add(c.Timeout))
	return c.TCPConn.Write(buf)
}

func DialUpstream(dialer Dialer, network, target string, timeout time.Duration) (*IdleTimeoutConn, error) {
	c, err := dialer.Dial(network, target)
	if err != nil {
		return nil, err
	}

	if tcpConn, ok := c.(*net.TCPConn); !ok {
		return nil, fmt.Errorf("not tcp conn")
	} else {
		return &IdleTimeoutConn{tcpConn, timeout}, nil
	}

}

func NewIdleTimeoutConn(c *net.TCPConn, timeout time.Duration) *IdleTimeoutConn {
	return &IdleTimeoutConn{c, timeout}
}
