package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/puckgo/network"
	"github.com/sirupsen/logrus"
)

// ServeConn to hanlde new conn request
type ServeConn func(conn net.Conn) error

// Proxy proxy for puckgo
type Proxy struct {
	Mode         config.Mode
	Timeout      time.Duration
	Listener     *net.TCPListener
	UpstreamAddr *net.TCPAddr
	network.Dialer
	conn.Reception
}

// NewProxy create a new proxy
func NewProxy(ma datasource.AccessList, proxyConfig *config.ProxyConfig) (*Proxy, error) {
	p := &Proxy{
		Mode: proxyConfig.Mode,
	}
	p.Timeout = time.Duration(proxyConfig.Timeout) * time.Second

	lnsa, err := net.ResolveTCPAddr("tcp", proxyConfig.Listen)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", lnsa)
	if err != nil {
		return nil, err
	}
	p.Listener = listener

	defer func() {
		if err != nil {
			p.Listener.Close()
		}
	}()

	if err = p.updateModeConfig(ma, proxyConfig); err != nil {
		return nil, err
	}

	return p, nil
}

// StartProxy start proxy to handle http and https
func (p *Proxy) StartProxy() {
	for {
		conn, err := p.Listener.AcceptTCP()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("accepting connection error")
			continue
		}

		go p.handleConn(conn)
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

	_, err := io.Copy(dst, src)
	return err
}

func (p *Proxy) handleConn(rawConn net.Conn) error {

	var (
		wg  = &sync.WaitGroup{}
		c   net.Conn
		err error
	)

	defer func() {
		rawConn.Close()
	}()

	timeout := p.Timeout

	addr, c, err := p.Recept(rawConn)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"downstream": rawConn.RemoteAddr(),
		}).Error("get request host from downstream failed")
		return err
	}

	src, err := conn.NewIdleTimeoutConn(c, timeout)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"downstream": rawConn.RemoteAddr(),
		}).Error("set timeout failed for downstream")
		return err
	}

	upstreamConn, err := p.Dial(addr)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"downstream": rawConn.RemoteAddr(),
		}).Error("dial upstream failed")
		return err
	}
	dst, err := conn.NewIdleTimeoutConn(upstreamConn, timeout)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"downstream": rawConn.RemoteAddr(),
		}).Error("set timeout failed for upstream")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"localsrc":   src.RemoteAddr(),
		"localdst":   src.LocalAddr(),
		"remotesrc":  dst.LocalAddr(),
		"remotedst":  dst.RemoteAddr(),
		"remotehost": addr.String(),
	}).Info("session info")

	wg.Add(2)

	go func() {
		if err := syncCopy(wg, dst, src); err != nil {
			logrus.WithFields(logrus.Fields{
				"error":      err.Error(),
				"upstream":   fmt.Sprintf("%s", addr.String()),
				"downstream": rawConn.RemoteAddr(),
			}).Error("copy from downstream to upstream failed")
		}
	}()

	go func() {
		if err := syncCopy(wg, src, dst); err != nil {
			logrus.WithFields(logrus.Fields{
				"error":      err.Error(),
				"upstream":   fmt.Sprintf("%s", addr.String()),
				"downstream": rawConn.RemoteAddr(),
			}).Error("copy from upstreamstream to downstream failed")
		}
	}()

	wg.Wait()
	return nil
}
