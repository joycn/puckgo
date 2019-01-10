package proxy

import (
	"fmt"
	"github.com/joycn/datasource"
	"github.com/joycn/dnsforward"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/puckgo/network"
)

func (p *Proxy) setTransparentMode(ma *datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	err := network.SetTransparentOpt(p.Listener)

	if proxyConfig.Transparent == nil {
		return fmt.Errorf("no transparent config found")
	}
	if err := network.ConfigTransparentNetwork(); err != nil {
		return err
	}
	f := dnsforward.NewDNSForwarder(ma)
	t := proxyConfig.Transparent
	go f.StartDNS(t.DefaultServer, t.SpecifiedServer, t.Listen)
	r := &conn.Transparent{DNS: f}
	s, err := conn.NewCryptoDialer("tcp4", proxyConfig.Upstream, proxyConfig.Key, true, ma)
	if err != nil {
		return err
	}
	p.Dialer = s
	p.Reception = r
	return nil
}

func (p *Proxy) updateModeConfig(ma *datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	switch p.Mode {
	case config.TransparentMode:
		return p.setTransparentMode(ma, proxyConfig)
	case config.SocksLocalMode:
		return p.setSocksLocalMode(ma, proxyConfig)
	case config.SocksServerMode:
		return p.setSocksServerMode(ma, proxyConfig)
	}
	return nil
}
