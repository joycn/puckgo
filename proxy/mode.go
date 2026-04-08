package proxy

import (
	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/socks"
)

func (p *Proxy) setSocksLocalMode(ma datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	config := &socks.Config{}
	r, err := socks.New(config)
	if err != nil {
		return err
	}
	if proxyConfig.Password != "" {
		p.Dialer, err = conn.NewCryptoDialer("tcp4", proxyConfig.Upstream, proxyConfig.Password, false, ma)
		if err != nil {
			return err
		}
	} else {
		p.Dialer, err = conn.NewNormalDialer("tcp4", proxyConfig.Upstream, false, ma)
		if err != nil {
			return err
		}
	}
	p.Reception = conn.NewMultiReception(r)
	return nil
}

func (p *Proxy) setSocksServerMode(ma datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	config := &socks.Config{}
	socks5, err := socks.New(config)
	if err != nil {
		return err
	}

	var r conn.Reception

	if proxyConfig.Password != "" {
		r, err = conn.NewCryptoReception(socks5, proxyConfig.Password)
		if err != nil {
			return err
		}
	} else {
		r, err = conn.NewNormalReception(socks5)
		if err != nil {
			return err
		}
	}
	s := &conn.DirectDialer{AccessList: ma, Match: true}
	p.Dialer = s
	p.Reception = r
	return nil
}

func (p *Proxy) setRelayServerMode(ma datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	config := &socks.Config{}
	socks5, err := socks.New(config)
	if err != nil {
		return err
	}

	var r conn.Reception
	if proxyConfig.Password != "" {
		r, err = conn.NewCryptoReception(socks5, proxyConfig.Password)
		if err != nil {
			return err
		}
	} else {
		r, err = conn.NewNormalReception(socks5)
		if err != nil {
			return err
		}
	}

	ps, err := conn.NewSocks5Dialer("tcp", proxyConfig.Upstream, proxyConfig.UpstreamUsername, proxyConfig.UpstreamPassword)
	if err != nil {
		return err
	}
	p.Dialer = ps
	p.Reception = r
	return nil
}

func (p *Proxy) updateModeConfig(ma datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	switch p.Mode {
	case config.SocksLocalMode:
		return p.setSocksLocalMode(ma, proxyConfig)
	case config.SocksServerMode:
		return p.setSocksServerMode(ma, proxyConfig)
	case config.RelayServerMode:
		return p.setRelayServerMode(ma, proxyConfig)
	}
	return nil
}
