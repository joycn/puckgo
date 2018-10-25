package cmd

import (
	"fmt"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/datasource"
	"github.com/joycn/puckgo/dnsforward"
	"github.com/joycn/puckgo/proxy"
	"github.com/sirupsen/logrus"
	"net/http"
	// net pprof
	_ "net/http/pprof"
)

func start(cfg *config.Config) error {
	logrus.SetFormatter(&logrus.TextFormatter{})
	lvl, err := logrus.ParseLevel(cfg.LogLevel)
	if err == nil {
		logrus.SetLevel(lvl)
	} else {
		fmt.Println("set log level failed: %s", err)
	}
	if cfg.LogLevel == "debug" {
		go func() {
			http.ListenAndServe(":6060", nil)
		}()
	}

	al, err := datasource.GetAccessList(cfg.DataSource)
	if err != nil {
		fmt.Println(err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"datasource": al,
	}).Debug("fetch access list success")

	go dnsforward.StartDNS(al, &cfg.DNS)
	proxy.StartProxy(al, &cfg.TransparentProxy)
	//proxy.StartSocks5Proxy(&cfg.Socks5Proxy)
	return nil
}
