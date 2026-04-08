package config

// Mode for proxy type
type Mode string

const (
	// SocksLocalMode socks5 local for standard socks server
	SocksLocalMode Mode = "local"
	// SocksServerMode socks5 server mode for private mode socks server
	SocksServerMode Mode = "server"
	// RelayServerMode relay server mode: accepts like server, forwards like local with split-tunnel
	RelayServerMode Mode = "relayServer"
)

// Config params for dnsforward
type Config struct {
	LogLevel   string
	DataSource string
	Listen     string
	Timeout    int
	Proxy      ProxyConfig
}

// ProxyProtocolMap protocol map to proxied
type ProxyProtocolMap map[string][]int

// ProxyConfig config for proxy
type ProxyConfig struct {
	Listen           string
	Upstream         string
	Timeout          int
	Password         string
	UpstreamUsername string
	UpstreamPassword string
	Mode
}

const (
	// DefaultProxyTimeout conn timeout for upstream
	DefaultProxyTimeout = 300
	// DefaultSocks5Listen default address listen for socks5 proxy
	DefaultSocks5Listen = ":1080"
)
