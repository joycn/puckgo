package conn

import (
	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/network"
	"github.com/joycn/socks"
	"net"
)

// Pac proxy auto config mode
type Pac struct {
	//CryptoDialer *CryptoDialer
	SpecDialer   network.Dialer
	DirectDialer network.Dialer
	datasource.AccessList
}

// Dial dial to proxy if match local list, otherwise dial directly
func (p *Pac) Dial(addr *socks.AddrSpec) (c net.Conn, err error) {
	if match := p.MatchDomain(addr.FQDN); match {
		return p.SpecDialer.Dial(addr)
	}
	return p.DirectDialer.Dial(addr)
}
