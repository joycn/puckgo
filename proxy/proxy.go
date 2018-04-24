package proxy

import (
	"bufio"
	"encoding/binary"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/puckgo/datasource"
	"github.com/joycn/puckgo/filter"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	//"github.com/joycn/puckgo/sni"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"syscall"
	"time"
	"unsafe"
)

var (
	proxyDialer proxy.Dialer
	filters     *filter.Filters
)

func createFilters(ma *datasource.AccessList) *filter.Filters {
	filters := filter.NewFilters(ma)
	httpFilter := filter.NewHTTPFilter()
	filters.AddFilter(httpFilter, uint16(80))
	filters.AddFilter(httpFilter, uint16(8081))
	httpsFilter := filter.NewHTTPSFilter()
	filters.AddFilter(httpsFilter, uint16(443))
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
func StartProxy(ma *datasource.AccessList, proxyMatch bool, tranparentProxyConfig *config.TransparentProxyConfig) {
	auth := &proxy.Auth{NoAuth: true}
	timeout := time.Duration(time.Duration(tranparentProxyConfig.ProxyTimeout) * time.Millisecond)
	filters = createFilters(ma)
	if tranparentProxyConfig.SecurityUpstream {
		proxyDialer, _ = PuckSocks("tcp", tranparentProxyConfig.ProxyUpstream, auth, conn.TLSDialer)
	} else {
		proxyDialer, _ = PuckSocks("tcp", tranparentProxyConfig.ProxyUpstream, auth, proxy.Direct)
	}

	lnsa, err := net.ResolveTCPAddr("tcp", tranparentProxyConfig.ProxyListen)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("parse listen address failed")
	}

	listener, err := net.ListenTCP("tcp", lnsa)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("tranparent proxy listen failed")
	}
	defer listener.Close()

	err = setTransparentOpt(listener)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("set tranparent failed")
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("accepting connection error")
			continue
		}
		go handleConn(conn, timeout, proxyMatch)
	}
}

func handleConn(rawConn *net.TCPConn, timeout time.Duration, proxyMatch bool) {

	var (
		wg   = &sync.WaitGroup{}
		buf  filter.Buffer
		host string
		port uint16
		d    proxy.Dialer
		err  error
	)

	c := conn.NewIdleTimeoutConn(rawConn, timeout)
	downstreamReader := bufio.NewReader(c)

	defer c.Close()

	//defer func() {
	////downstreamReader.Reset(nil)
	//if needCloseConn {
	//if err := c.Close(); err != nil {
	//logrus.WithFields(logrus.Fields{
	//"error":  err.Error(),
	//"remote": c.RemoteAddr(),
	//}).Error("conn close failed")
	//}
	//}
	//}()

	//c.SetLinger(0)
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
		//host, port, err = getOriginalDst(rawConn)
		var ports string
		host, ports, err = net.SplitHostPort(rawConn.LocalAddr().String())

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("get client address failed")
			return
		}
		portint, err := strconv.Atoi(ports)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("get client port failed")
			return
		}
		port = uint16(portint)
		var domainName string
		domainName, buf, err = filters.ExecFilters(downstreamReader, port)
		if err != nil {
			// do something
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
				"dport": port,
			}).Warning("exec filters failed")
		} else {
			if domainName != "" {
				host = domainName
			}
		}

	}
	matched := filters.Match(host)

	if matched == proxyMatch {
		d = proxyDialer
	} else {
		d = proxy.Direct
	}

	upstream := fmt.Sprintf("%s:%d", host, port)
	sendConn, err := conn.DialUpstream(d, "tcp", upstream, timeout)

	//sendConn.SetLinger(0)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":    err.Error(),
			"upstream": upstream,
		}).Error("dial upstream failed")
		return
	}

	wg.Add(1)

	go copyData(wg, c, sendConn)

	//defer c.CloseRead()
	defer sendConn.Close()
	if buf != nil {
		buf.Write(sendConn)
	}
	if _, err := io.Copy(sendConn, downstreamReader); err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"upstream":   upstream,
			"downstream": c.RemoteAddr(),
		}).Error("copy from downstream to upstream failed")
	}

	sendConn.CloseWrite()

	wg.Wait()
}

func getsockopt(s int, level int, name int, val uintptr, vallen *uint32) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(name), uintptr(val), uintptr(unsafe.Pointer(vallen)), 0)
	if e1 != 0 {
		err = e1
	}
	return
}

func copyData(wg *sync.WaitGroup, dst, src *conn.IdleTimeoutConn) {

	defer wg.Done()

	if _, err := io.Copy(dst, src); err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"upstream":   src.RemoteAddr(),
			"downstream": dst.RemoteAddr(),
		}).Error("copy from upstream to downstream failed")
	}
	dst.CloseWrite()
}

//func handleHTTP(r *bufio.Reader) (*http.Request, error) {

//fmt.Println("http", time.Now())

//req, err := http.ReadRequest(r)
//if err != nil {
//fmt.Println(err)
//return nil, err
//}

//fmt.Println(req.Host)
//return req, nil
//}

func getOriginalDst(clientConn *net.TCPConn) (string, uint16, error) {
	clientConnFile, err := clientConn.File()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"downstream": clientConn.RemoteAddr(),
		}).Error("get client conn file")
		return "", 0, err
	}
	defer clientConnFile.Close()

	//sa, err := syscall.GetsockoptInet4Addr(int(clientConnFile.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)

	var sa syscall.RawSockaddrInet4
	var size = uint32(unsafe.Sizeof(sa))

	err = getsockopt(int(clientConnFile.Fd()), syscall.IPPROTO_IP, 80, uintptr(unsafe.Pointer(&sa)), &size)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err.Error(),
			"downstream": clientConn.RemoteAddr(),
		}).Error("get origin dst")
		return "", 0, err
	}

	addr := net.IPv4(sa.Addr[0], sa.Addr[1], sa.Addr[2], sa.Addr[3]).String()
	nport := sa.Port
	lport := binary.BigEndian.Uint16((*(*[2]byte)(unsafe.Pointer(&nport)))[:])
	port := uint16(lport)

	local := fmt.Sprintf("%s:%d", addr, port)

	if local == clientConn.LocalAddr().String() {
		return addr, port, fmt.Errorf("no nat request")
	}

	return addr, port, nil

	//return addr + ":" + port, nil
}

//func handleHTTPS(b *bufio.Reader) (io.Reader, error) {

//fmt.Println("https", time.Now())

//_, err := sni.ReadServerName(b)
//if err != nil {
//return b, nil
//} else {
//return nil, err
//}
//}
