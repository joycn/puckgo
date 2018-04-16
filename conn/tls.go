package conn

import (
	"crypto/tls"
	"net"
)

// TLSDialer default tls dialer
var TLSDialer = tlsDialer{}

// tlsDialer socks5 dialer with tls support
type tlsDialer struct {
}

// Dial dial upstream with socks over tls
func (dialer tlsDialer) Dial(network, addr string) (net.Conn, error) {
	config := &tls.Config{InsecureSkipVerify: true, ClientSessionCache: tls.NewLRUClientSessionCache(200)}
	return tls.Dial(network, addr, config)
}
