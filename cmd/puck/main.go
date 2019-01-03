package main

import (
	"github.com/armon/go-socks5"
	"os"
)

func main() {
	var conf *socks5.Config
	conf = &socks5.Config{NoReply: true}

	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	go startDNS(":53", 3)

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.ListenAndServe("tcp4", os.Args[1]); err != nil {
		panic(err)
	}
}
