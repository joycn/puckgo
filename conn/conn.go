package conn

import (
	"fmt"
	"net"
	"time"
)

// Dialer is a generic dialer to dial with different protocols
type Dialer interface {
	// Dial connects to the given address via the proxy.
	Dial(network, addr string) (c net.Conn, err error)
}

// IdleTimeoutConn is a tcp conn and reset deadline every read and write
type IdleTimeoutConn struct {
	*net.TCPConn
	Timeout time.Duration
}

// Read set deadline before every read
func (c *IdleTimeoutConn) Read(buf []byte) (int, error) {
	c.SetDeadline(time.Now().Add(c.Timeout))
	return c.TCPConn.Read(buf)
}

// Write set deadline before every write
func (c *IdleTimeoutConn) Write(buf []byte) (int, error) {
	c.SetDeadline(time.Now().Add(c.Timeout))
	return c.TCPConn.Write(buf)
}

// DialUpstream dial upstream with dialer and return an IdleTimeoutConn
func DialUpstream(dialer Dialer, network, target string, timeout time.Duration) (*IdleTimeoutConn, error) {
	c, err := dialer.Dial(network, target)
	if err != nil {
		return nil, err
	}

	var tcpConn *net.TCPConn
	var ok bool
	if tcpConn, ok = c.(*net.TCPConn); !ok {
		return nil, fmt.Errorf("not tcp conn")
	}
	return &IdleTimeoutConn{tcpConn, timeout}, nil

}

// NewIdleTimeoutConn create a new IdleTimeoutConn with timeout
func NewIdleTimeoutConn(c *net.TCPConn, timeout time.Duration) *IdleTimeoutConn {
	return &IdleTimeoutConn{c, timeout}
}
