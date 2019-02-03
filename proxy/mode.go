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
	var ps conn.Dialer
	if proxyConfig.Key != "" {
		ps, err = conn.NewCryptoDialer("tcp4", proxyConfig.Upstream, proxyConfig.Key, false, ma)
		if err != nil {
			return err
		}
	} else {
		ps, err = conn.NewNormalDialer("tcp4", proxyConfig.Upstream, false, ma)
		if err != nil {
			return err
		}
	}
	ds := &conn.DirectDialer{}
	p.Dialer = &conn.Pac{ps, ds, ma}
	p.Reception = r
	return nil
}
