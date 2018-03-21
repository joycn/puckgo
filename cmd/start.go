package cmd

import (
	//"github.com/Sirupsen/logrus"
	"fmt"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/datasource"
	"github.com/joycn/puckgo/dnsforward"
	"github.com/joycn/puckgo/proxy"
	"time"
)

func start(cfg *config.Config) error {

	ma, err := datasource.GetMatchActions(cfg.DataSource)
	if err != nil {
		fmt.Println(err)
		return err
	}
	go dnsforward.StartDNS(ma, cfg.DefaultServer, cfg.ExceptiveServer, cfg.Listen, false)
	timeout := time.Duration(time.Duration(cfg.ProxyTimeout) * time.Millisecond)
	proxy.StartProxy(ma, cfg.ProxyListen, cfg.ProxyUpstream, timeout)
	return nil
}
