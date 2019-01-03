package config

import (
	"github.com/joycn/dnsforward"
)

// PublicService whether run as a public service mode or a local gw mode
var PublicService bool

// Config params for dnsforward
type Config struct {
	LogLevel         string
	DataSource       string
	DNS              dnsforward.DNSConfig
	TransparentProxy TransparentProxyConfig
}

// ProxyProtocolMap protocol map to proxied
type ProxyProtocolMap map[string][]int

// TransparentProxyConfig config params for transparent proxy
type TransparentProxyConfig struct {
	Listen   string
	Upstream string
	Bind     string
	Timeout  int
}

const (
	//DefaultDataSource use /usr/loca/etc/dns/forward/datasource as url list by default
	DefaultDataSource = "file:/usr/local/etc/dns/forward/datasource"
	// DefaultServer DNS server for DNS in China
	DefaultServer = "114.114.114.114:53"
	// SpecifiedServer DNS server for DNS outside China
	SpecifiedServer = "8.8.8.8:53"
	// DefaultListen listen address and port
	DefaultListen = "0.0.0.0:53"
	// DefaultProxyListen local transparent proxy listen on
	DefaultProxyListen = ":1200"
	// DefaultProxyTimeout conn timeout for upstream
	DefaultProxyTimeout = 3000
	// DefaultSocks5Listen default address listen for socks5 proxy
	DefaultSocks5Listen = ":1080"
)
