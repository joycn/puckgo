package conn

import (
	"fmt"
	"io"
	"net"
	"time"
)

const (
	// FinTimeout timeout in seconds after rcv fin packet
	FinTimeout = 30
)

// Type UpstreamType
type Type string

const (
	// UDP direct udp proxy
	UDP Type = "udp"
	// TCP direct tcp proxy
	TCP Type = "tcp"
	// Socks5 socks5 proxy
	Socks5 Type = "socks5"
	// SecuritySocks5 socks5 over ts proxy
	SecuritySocks5 Type = "ssocks5"
	// Quic socks5 over quic proxy
	Quic Type = "quic"
)

// Peer local or remote address and type
type Peer struct {
	Address string
	Type    string
}

// Dialer is a generic dialer to dial with different protocols
type Dialer interface {
	// Dial connects to the given address via the proxy.
	Dial(network, addr string) (c net.Conn, err error)
}

// StreamConn net.Conn plus CloseWrite for tcp and tls conn
type StreamConn interface {
	io.Reader
	// Write writes data to the stream.
	// Write can be made to time out and return a net.Error with Timeout() == true
	// after a fixed time limit; see SetDeadline and SetWriteDeadline.
	// If the stream was canceled by the peer, the error implements the StreamError
	// interface, and Canceled() == true.
	io.Writer
	// Close closes the write-direction of the stream.
	// Future calls to Write are not permitted after calling Close.
	// It must not be called concurrently with Write.
	// It must not be called after calling CancelWrite.
	io.Closer
	// SetReadDeadline sets the deadline for future Read calls and
	// any currently-blocked Read call.
	// A zero value for t means Read will not time out.
	SetReadDeadline(t time.Time) error
	// SetWriteDeadline sets the deadline for future Write calls
	// and any currently-blocked Write call.
	// Even if write times out, it may return n > 0, indicating that
	// some of the data was successfully written.
	// A zero value for t means Write will not time out.
	SetWriteDeadline(t time.Time) error
	// SetDeadline sets the read and write deadlines associated
	// with the connection. It is equivalent to calling both
	// SetReadDeadline and SetWriteDeadline.
	SetDeadline(t time.Time) error
	// CloseWrite shuts down the writing side of the Stream connection.
	// Most callers should just use Close.
	CloseWrite() error
	//// CloseRead shuts down the reading side of the Stream connection.
	//// Most callers should just use Close.
	//CloseRead() error
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
