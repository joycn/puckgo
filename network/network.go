package network

import (
	"github.com/joycn/socks"
	"net"
)

// Dialer function type for dial
type Dialer interface {
	Dial(addr *socks.AddrSpec) (c net.Conn, err error)
}
