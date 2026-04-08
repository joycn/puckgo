package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/proxy"
	"github.com/sirupsen/logrus"
	// net pprof
	//_ "net/http/pprof"
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

	var al *datasource.AccessList
	if strings.HasPrefix(cfg.DataSource, "pass://") {
		al = &datasource.AccessList{
			Domains: make(datasource.DomainMap),
			Subnets: make(datasource.SubnetMap),
			RWMutex: &sync.RWMutex{},
		}
	} else {
		var err error
		al, err = datasource.GetAccessList(cfg.DataSource)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	logrus.WithFields(logrus.Fields{
		"datasource": al,
	}).Debug("fetch access list success")

	p, err := proxy.NewProxy(*al, &cfg.Proxy)
	if err != nil {
		fmt.Println(err)
		return err
	}
	p.StartProxy()
	return nil
}
