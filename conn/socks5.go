package conn

import (
	"fmt"
	"net"

	"github.com/joycn/socks"
	"golang.org/x/net/proxy"
)

// Socks5Dialer dials upstream via a standard SOCKS5 proxy
type Socks5Dialer struct {
	dialer proxy.Dialer
}

// NewSocks5Dialer creates a Socks5Dialer using golang.org/x/net/proxy.SOCKS5
func NewSocks5Dialer(network, upstream, username, password string) (*Socks5Dialer, error) {
	var auth *proxy.Auth
	if username != "" || password != "" {
		auth = &proxy.Auth{User: username, Password: password}
	}
	d, err := proxy.SOCKS5(network, upstream, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}
	return &Socks5Dialer{dialer: d}, nil
}

// Dial connects to the target address via the SOCKS5 proxy
func (d *Socks5Dialer) Dial(addr *socks.AddrSpec) (net.Conn, error) {
	return d.dialer.Dial("tcp", addr.String())
}
