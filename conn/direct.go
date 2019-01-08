package conn

import (
	"github.com/joycn/socks"
	"net"
)

// DirectDialer dial target
type DirectDialer struct {
}

// Dial target and return a net.Conn
func (d *DirectDialer) Dial(addr *socks.AddrSpec) (c net.Conn, err error) {
	return net.Dial("tcp4", addr.String())
}
