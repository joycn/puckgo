package config

// Config params for dnsforward
type Config struct {
	LogLevel         string
	DataSource       string
	Listen           string
	DefaultAction    bool
	DefaultServer    string
	ExceptiveServer  string
	ProxyListen      string
	ProxyUpstream    string
	ProxyTimeout     int
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
)
