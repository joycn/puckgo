package conn

import (
	"github.com/joycn/datasource"
	"github.com/joycn/socks"
	"net"
)

// Pac proxy auto config mode
type Pac struct {
	CryptoDialer *CryptoDialer
	DirectDialer *DirectDialer
	*datasource.AccessList
}

// Dial dial to proxy if match local list, otherwise dial directly
func (p *Pac) Dial(addr *socks.AddrSpec) (c net.Conn, err error) {
	if match := p.MatchDomain(addr.FQDN); match {
		return p.CryptoDialer.Dial(addr)
	}
	return p.DirectDialer.Dial(addr)
}
