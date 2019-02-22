package conn

import (
	"fmt"
	"github.com/joycn/socks"
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

// Reception function type to get request target
type Reception interface {
	Recept(c net.Conn) (*socks.AddrSpec, net.Conn, error)
}

// StreamConn net.Conn plus CloseWrite for tcp and tls conn
type StreamConn interface {
	net.Conn
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
//func DialUpstream(dialer Dialer, addr *socks.AddrSpec, timeout time.Duration) (*IdleTimeoutConn, error) {
//c, err := dial(addr)
//if err != nil {
//return nil, err
//}

//var streamConn StreamConn
//var ok bool

//if streamConn, ok = c.(StreamConn); !ok {
//return nil, fmt.Errorf("not stream conn")
//}

//return &IdleTimeoutConn{streamConn, timeout}, nil

//}

// NewIdleTimeoutConn create a new IdleTimeoutConn with timeout
func NewIdleTimeoutConn(c net.Conn, timeout time.Duration) (*IdleTimeoutConn, error) {
	s, ok := c.(StreamConn)
	if !ok {
		return nil, fmt.Errorf("not streamConn type")
	}
	return &IdleTimeoutConn{s, timeout}, nil
}
