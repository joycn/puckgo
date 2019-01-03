package proxy

import (
	//"crypto/tls"
	"github.com/joycn/dnsforward"
	"github.com/joycn/dnsforward/datasource"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/puckgo/filter"
	"github.com/joycn/puckgo/network"
	"github.com/sirupsen/logrus"
	"sync"
	//"github.com/joycn/puckgo/sni"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"syscall"
	"time"
)

var (
	proxyDialer proxy.Dialer
	filters     *filter.Filters
)

func createFilters(ma *datasource.AccessList, pm config.ProxyProtocolMap) *filter.Filters {
	filters := filter.NewFilters(ma)
	for name, ports := range pm {
		for _, port := range ports {
			filters.AddFilter(name, port)
		}
	}
	return filters
}

func setTransparentOpt(l *net.TCPListener) error {
	cs, err := l.File()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("get listener file failed")
		return err
	}

	defer cs.Close()

	return syscall.SetsockoptInt(int(cs.Fd()), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
}

// StartProxy start proxy to handle http and https
func StartProxy(ma *datasource.AccessList, tranparentProxyConfig *config.TransparentProxyConfig) {
	var err error
	timeout := time.Duration(time.Duration(tranparentProxyConfig.Timeout) * time.Millisecond)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("dial upstream proxy failed")
	}

	lnsa, err := net.ResolveTCPAddr("tcp", tranparentProxyConfig.Listen)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("parse listen address failed")
	}

	if !config.PublicService {
		if err := network.ConfigTransparentNetwork(); err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Fatal("route config failed")
		}
	}

	listener, err := net.ListenTCP("tcp", lnsa)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("tranparent proxy listen failed")
	}
	defer listener.Close()

	err = network.SetTransparentOpt(listener)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("set tranparent failed")
	}

	target, err := net.ResolveTCPAddr("tcp", tranparentProxyConfig.Upstream)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("proxy upstream format error")
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("accepting connection error")
			continue
		}

		go handleConn(conn, timeout, target)
	}
}

type bidirectionalConn interface {
	io.Reader
	io.Writer
	CloseWrite() error
}

func syncCopy(wg *sync.WaitGroup, dst, src bidirectionalConn) error {
	defer wg.Done()
	defer dst.CloseWrite()
	//defer src.CloseRead()

	_, err := io.Copy(dst, src)
	return err
}

func handleConn(rawConn *net.TCPConn, timeout time.Duration, target *net.TCPAddr) {

	var (
		wg   = &sync.WaitGroup{}
		host string
		port int
		err  error
	)

	src := conn.NewIdleTimeoutConn(rawConn, timeout)

	defer src.Close()

	if config.PublicService {
		host, port, err = HandleSocks5Request(rawConn)
		if err != nil {
			// do something
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("handle socks5 request")
			return
		}
	} else {
		dst := rawConn.LocalAddr().(*net.TCPAddr)
		ipAddress := dst.IP.String()

		host = dnsforward.GetDomain(ipAddress)

		port = dst.Port
	}

	stream, err := net.DialTCP("tcp4", nil, target)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("open stream failed")
		return
	}
	dst := conn.NewIdleTimeoutConn(stream, timeout)
	defer dst.Close()

	if err = Connect(dst, host, port); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("connect upstream failed")
		return
	}

	wg.Add(2)

	go func() {
		if err := syncCopy(wg, dst, src); err != nil {
			logrus.WithFields(logrus.Fields{
				"error":      err.Error(),
				"upstream":   fmt.Sprintf("%s:%d", host, port),
				"downstream": rawConn.RemoteAddr(),
			}).Error("copy from downstream to upstream failed")
		}
	}()

	go func() {
		if err := syncCopy(wg, src, dst); err != nil {
			logrus.WithFields(logrus.Fields{
				"error":      err.Error(),
				"upstream":   fmt.Sprintf("%s:%d", host, port),
				"downstream": rawConn.RemoteAddr(),
			}).Error("copy from upstreamstream to downstream failed")
		}
	}()

	wg.Wait()
	src.Close()
}
