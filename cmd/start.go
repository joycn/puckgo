package cmd

import (
	"fmt"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/datasource"
	"github.com/joycn/puckgo/dnsforward"
	"github.com/joycn/puckgo/proxy"
	"github.com/sirupsen/logrus"
	"time"
)

func start(cfg *config.Config) error {
	logrus.SetFormatter(&logrus.TextFormatter{})
	al, err := datasource.GetAccessList(cfg.DataSource)
	if err != nil {
		fmt.Println(err)
		return err
	}
	go dnsforward.StartDNS(al, cfg.DefaultServer, cfg.ExceptiveServer, cfg.Listen, false)
	timeout := time.Duration(time.Duration(cfg.ProxyTimeout) * time.Millisecond)
	proxy.StartProxy(al, cfg.ProxyListen, cfg.ProxyUpstream, timeout, cfg.SecurityUpstream)
	return nil
}
