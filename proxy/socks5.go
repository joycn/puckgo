package proxy

import (
	"github.com/armon/go-socks5"
	"github.com/joycn/puckgo/config"
	"github.com/sirupsen/logrus"
)

// StartSocks5Proxy start a socks proxy on listen address
func StartSocks5Proxy(socks5ProxyConfig *config.Socks5ProxyConfig) {
	var conf *socks5.Config
	//if len(os.Args) > 1 {
	//cert, err := tls.LoadX509KeyPair(os.Args[1], os.Args[2])
	//if err != nil {
	//log.Fatalf("server: loadkeys: %s", err)
	//}
	//config = &tls.Config{Certificates: []tls.Certificate{cert}}
	//config.Rand = rand.Reader
	//}
	//conf = &socks5.Config{TLSConfig: config}
	conf = &socks5.Config{}

	server, err := socks5.New(conf)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("sock5 proxy config error")
	}

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.ListenAndServe("tcp", socks5ProxyConfig.Socks5Listen); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("start socks5 proxy failed")
	}
}
