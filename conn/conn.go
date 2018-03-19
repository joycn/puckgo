package conn

import (
	"net"
	"time"
)

type Dialer interface {
	// Dial connects to the given address via the proxy.
	Dial(network, addr string) (c net.Conn, err error)
}

type IdleTimeoutConn struct {
	net.Conn
	Timeout time.Duration
}

func (c *IdleTimeoutConn) Read(buf []byte) (int, error) {
	c.SetDeadline(time.Now().Add(c.Timeout))
	return c.Conn.Read(buf)
}

func (c *IdleTimeoutConn) Write(buf []byte) (int, error) {
	c.SetDeadline(time.Now().Add(c.Timeout))
	return c.Conn.Write(buf)
}

func DialUpstream(dialer Dialer, network, target string, timeout time.Duration) (*IdleTimeoutConn, error) {
	c, err := dialer.Dial(network, target)
	if err != nil {
		return nil, err
	}

	return &IdleTimeoutConn{c, timeout}, nil
}

func NewIdleTimeoutConn(c net.Conn, timeout time.Duration) *IdleTimeoutConn {
	return &IdleTimeoutConn{c, timeout}
}
