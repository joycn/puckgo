package proxy

import (
	"fmt"
	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/puckgo/dnsforward"
	"github.com/joycn/puckgo/network"
	"github.com/joycn/socks"
)

func (p *Proxy) setTransparentMode(ma *datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	err := network.SetTransparentOpt(p.Listener)

	if proxyConfig.DNSConfig == nil {
		return fmt.Errorf("no transparent config found")
	}
	if err := network.ConfigTransparentNetwork(); err != nil {
		return err
	}

	var s network.Dialer
	if proxyConfig.Key != "" {
		s, err = conn.NewCryptoDialer("tcp4", proxyConfig.Upstream, proxyConfig.Key, true, ma)
		if err != nil {
			return err
		}
	} else {
		s, err = conn.NewNormalDialer("tcp4", proxyConfig.Upstream, true, ma)
		if err != nil {
			return err
		}
	}
	dnsConfig := proxyConfig.DNSConfig
	f := dnsforward.NewDNSForwarder(dnsConfig.Listen, dnsConfig.SpecifiedServer, ma, s)
	go f.StartDNS()
	r := &conn.Transparent{DNS: f}
	p.Dialer = s
	p.Reception = r
	return nil
}

func (p *Proxy) setSocksServerMode(ma *datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	//f := dnsforward.NewDNSForwarder(ma)
	//dnsConfig := proxyConfig.DNSConfig
	//go f.StartDNS("", dnsConfig.SpecifiedServer, dnsConfig.Listen)

	config := &socks.Config{}
	socks5, err := socks.New(config)
	if err != nil {
		return err
	}

	var r conn.Reception

	if proxyConfig.Key != "" {
		r, err = conn.NewCryptoReception(socks5, proxyConfig.Key)
		if err != nil {
			return err
		}
	} else {
		r, err = conn.NewNormalReception(socks5)
		if err != nil {
			return err
		}
	}
	s := &conn.DirectDialer{ma, true}
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
