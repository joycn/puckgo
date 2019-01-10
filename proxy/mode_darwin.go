package proxy

import (
	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/config"
)

func (p *Proxy) updateModeConfig(ma *datasource.AccessList, proxyConfig *config.ProxyConfig) error {
	switch p.Mode {
	case config.SocksLocalMode:
		return p.setSocksLocalMode(ma, proxyConfig)
	case config.SocksServerMode:
		return p.setSocksServerMode(ma, proxyConfig)
	}
	return nil
}
