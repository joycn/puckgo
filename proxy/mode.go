package proxy

import (
	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/socks"
)

func (p *Proxy) setSocksLocalMode(ma *datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	config := &socks.Config{}
	r, err := socks.New(config)
	if err != nil {
		return err
	}
	ps, err := conn.NewCryptoDialer("tcp4", proxyConfig.Upstream, proxyConfig.Key, false, ma)
	if err != nil {
		return err
	}
	ds := &conn.DirectDialer{}
	p.Dialer = &conn.Pac{ps, ds, ma}
	p.Reception = r
	return nil
}

func (p *Proxy) setSocksServerMode(ma *datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	config := &socks.Config{}
	socks5, err := socks.New(config)
	if err != nil {
		return err
	}

	r, err := conn.NewCryptoReception(socks5, proxyConfig.Key)
	if err != nil {
		return err
	}
	s := &conn.DirectDialer{ma, true}
	p.Dialer = s
	p.Reception = r
	return nil
}
