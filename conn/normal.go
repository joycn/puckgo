package conn

import (
	"fmt"
	"github.com/joycn/datasource"
	"github.com/joycn/socks"
	"net"
)

// NormalDialer dial target and return a cipher conn
type NormalDialer struct {
	Network      string
	UpstreamAddr string
	match        bool
	*datasource.AccessList
}

// NewNormalDialer create NormalDialer with a base64 string
func NewNormalDialer(network, upstream, key string, match bool, ma *datasource.AccessList) (*NormalDialer, error) {
	return &NormalDialer{Network: network, UpstreamAddr: upstream, AccessList: ma, match: match}, nil
}

// Dial target and return a cipher conn
func (d *NormalDialer) Dial(addr *socks.AddrSpec) (c net.Conn, err error) {
	if d.match {
		if match := d.MatchDomain(addr.FQDN); !match {
			return nil, fmt.Errorf("FQDN not match")
		}
	}

	if c, err = net.Dial(d.Network, d.UpstreamAddr); err != nil {
		return nil, err
	}
	if err = socks.Connect(c, addr, true); err != nil {
		return nil, err
	}

	return c, nil
}

// NormalReception get target ip and port and return a NormalConn
type NormalReception struct {
	s *socks.Server
}

// NewNormalReception return a NormalReception with noauth socks server
func NewNormalReception(s *socks.Server, key string) (*NormalReception, error) {
	r := &NormalReception{s: s}
	r.s.NoAuth = true
	return r, nil
}

// Recept get target ip and port and return a NormalConn
func (cr *NormalReception) Recept(c net.Conn) (*socks.AddrSpec, net.Conn, error) {
	addr, _, err := cr.s.Recept(c)
	if err != nil {
		return nil, nil, err
	}
	return addr, c, nil
}
