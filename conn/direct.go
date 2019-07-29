package conn

import (
	"fmt"
	"github.com/joycn/datasource"
	"github.com/joycn/socks"
	"net"
)

// DirectDialer dial target
type DirectDialer struct {
	datasource.AccessList
	Match bool
}

// Dial target and return a net.Conn
func (d *DirectDialer) Dial(addr *socks.AddrSpec) (c net.Conn, err error) {
	if d.Match {
		if match := d.MatchDomain(addr.FQDN); !match {
			return nil, fmt.Errorf("FQDN not match %s", addr.FQDN)
		}
	}
	return net.Dial("tcp4", addr.String())
}
