package conn

import (
	"github.com/joycn/dnsforward"
	"github.com/joycn/socks"
	"net"
)

// Transparent get target from conn.Conn
type Transparent struct {
	DNS *dnsforward.DNSForwarder
}

// Recept get real request from net.Conn
func (t *Transparent) Recept(c net.Conn) (*socks.AddrSpec, net.Conn, error) {
	dst := c.LocalAddr().(*net.TCPAddr)

	ipAddress := dst.IP.String()
	addr := &socks.AddrSpec{Port: dst.Port}
	host, found := t.DNS.GetDomain(ipAddress)
	if found {
		addr.FQDN = host
	}
	return addr, c, nil

}
