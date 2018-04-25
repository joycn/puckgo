package cmd

import (
	"fmt"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/datasource"
	"github.com/joycn/puckgo/dnsforward"
	"github.com/joycn/puckgo/proxy"
	"github.com/sirupsen/logrus"
)

func start(cfg *config.Config) error {
	logrus.SetFormatter(&logrus.TextFormatter{})
	al, err := datasource.GetAccessList(cfg.DataSource)
	if err != nil {
		fmt.Println(err)
		return err
	}

	go dnsforward.StartDNS(al, cfg.ProxyMatch, &cfg.DNS)
	proxy.StartProxy(al, cfg.ProxyMatch, &cfg.TransparentProxy)
	//proxy.StartSocks5Proxy(&cfg.Socks5Proxy)
	return nil
}
