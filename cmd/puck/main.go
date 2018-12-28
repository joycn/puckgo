package main

import (
	"crypto/rand"
	"crypto/tls"
	"github.com/armon/go-socks5"
	"log"
	"os"
)

func main() {
	var tlsConf *tls.Config
	var conf *socks5.Config
	if len(os.Args) > 3 {
		cert, err := tls.LoadX509KeyPair(os.Args[2], os.Args[3])
		if err != nil {
			log.Fatalf("server: loadkeys: %s", err)
		}
		tlsConf = &tls.Config{Certificates: []tls.Certificate{cert}, ClientSessionCache: tls.NewLRUClientSessionCache(200)}
		tlsConf.Rand = rand.Reader
	}
	conf = &socks5.Config{}

	server, err := socks5.NewQocks(conf, tlsConf)
	if err != nil {
		panic(err)
	}

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.ListenAndServe(os.Args[1]); err != nil {
		panic(err)
	}
}
