package cmd

import (
	//"github.com/Sirupsen/logrus"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/dnsforward"
	"github.com/joycn/puckgo/proxy"
	"time"
)

func start(cfg *config.Config) error {
	go dnsforward.StartDNS(cfg.DataSource, cfg.DefaultServer, cfg.ExceptiveServer, cfg.Listen, false)
	timeout := time.Duration(time.Duration(cfg.ProxyTimeout) * time.Millisecond)
	proxy.StartProxy(cfg.ProxyListen, cfg.ProxyUpstream, timeout)
	return nil
}
