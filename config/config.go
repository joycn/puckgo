package config

// PublicService whether run as a public service mode or a local gw mode
var PublicService bool

// Config params for dnsforward
type Config struct {
	LogLevel         string
	DataSource       string
	ProxyMatch       bool
	DNS              DNSConfig
	TransparentProxy TransparentProxyConfig
	Socks5Proxy      Socks5ProxyConfig
}

// DNSConfig dns config params for dns server
type DNSConfig struct {
	Listen          string
	DefaultServer   string
	ExceptiveServer string
}

// TransparentProxyConfig config params for transparent proxy
type TransparentProxyConfig struct {
	ProxyListen string
	//ProxyListen      []string
	ProxyUpstream    string
	ProxyTimeout     int
	SecurityUpstream bool
}

// Socks5ProxyConfig config params for socks5 proxy
type Socks5ProxyConfig struct {
	Socks5Listen     string
	SecurityUpstream bool
}

const (
	//DefaultDataSource use /usr/loca/etc/dns/forward/datasource as url list by default
	DefaultDataSource = "file:/usr/local/etc/dns/forward/datasource"
	// DefaultServer DNS server for DNS in China
	DefaultServer = "114.114.114.114:53"
	// ExceptiveServer DNS server for DNS outside China
	ExceptiveServer = "8.8.8.8:53"
	// DefaultListen listen address and port
	DefaultListen = "0.0.0.0:53"
	// DefaultProxyListen local transparent proxy listen on
	DefaultProxyListen = ":1200"
	// DefaultProxyTimeout conn timeout for upstream
	DefaultProxyTimeout = 3000
	// DefaultSocks5Listen default address listen for socks5 proxy
	DefaultSocks5Listen = ":1080"
)
