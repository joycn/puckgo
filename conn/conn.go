package conn

import (
	"fmt"
	"net"
	"time"
)

const (
	// FinTimeout timeout in seconds after rcv fin packet
	FinTimeout = 30
)

// Dialer is a generic dialer to dial with different protocols
type Dialer interface {
	// Dial connects to the given address via the proxy.
	Dial(network, addr string) (c net.Conn, err error)
}

// StreamConn net.Conn plus CloseWrite for tcp and tls conn
type StreamConn interface {
	net.Conn
	CloseWrite() error
}

// IdleTimeoutConn is a tcp conn and reset deadline every read and write
type IdleTimeoutConn struct {
	StreamConn
	Timeout time.Duration
}

// Read set deadline before every read
func (c *IdleTimeoutConn) Read(buf []byte) (int, error) {
	if c.Timeout != 0 {
		c.SetDeadline(time.Now().Add(c.Timeout))
	}
	return c.StreamConn.Read(buf)
}

// Write set deadline before every write
func (c *IdleTimeoutConn) Write(buf []byte) (int, error) {
	if c.Timeout != 0 {
		c.SetDeadline(time.Now().Add(c.Timeout))
	}
	return c.StreamConn.Write(buf)
}

// DialUpstream dial upstream with dialer and return an IdleTimeoutConn
func DialUpstream(dialer Dialer, network, target string, timeout time.Duration) (*IdleTimeoutConn, error) {
	c, err := dialer.Dial(network, target)
	if err != nil {
		return nil, err
	}

	var streamConn StreamConn
	var ok bool

	if streamConn, ok = c.(StreamConn); !ok {
		return nil, fmt.Errorf("not stream conn")
	}

	return &IdleTimeoutConn{streamConn, timeout}, nil

}

// NewIdleTimeoutConn create a new IdleTimeoutConn with timeout
func NewIdleTimeoutConn(c StreamConn, timeout time.Duration) *IdleTimeoutConn {
	return &IdleTimeoutConn{c, timeout}
}
