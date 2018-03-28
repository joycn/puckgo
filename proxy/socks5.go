package proxy

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/joycn/puckgo/conn"
	"github.com/joycn/puckgo/datasource"
	"github.com/joycn/puckgo/filter"
	//"github.com/joycn/puckgo/sni"
	"golang.org/x/net/proxy"
	"io"
	"net"
	//"net/http"
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
	filters.AddFilter(httpFilter)
	httpsFilter := filter.NewHTTPSFilter()
	filters.AddFilter(httpsFilter)
	return filters
}

// StartProxy start proxy to handle http and https
func StartProxy(ma *datasource.AccessList, listen, upstream string, timeout time.Duration) {

	filters = createFilters(ma)
	proxyDialer, _ = proxy.SOCKS5("tcp", upstream, nil, proxy.Direct)

	lnsa, err := net.ResolveTCPAddr("tcp", listen)
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenTCP("tcp", lnsa)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go handleConn(conn, timeout)
	}
}

func handleConn(rawConn *net.TCPConn, timeout time.Duration) {

	var (
		needProxied bool
		buf         filter.Buffer
		host        string
		port        uint16
		d           proxy.Dialer
		err         error
	)

	c := conn.NewIdleTimeoutConn(rawConn, timeout)
	downstreamReader := bufio.NewReader(c)

	defer func() {
		//downstreamReader.Reset(nil)
		if err := c.Close(); err != nil {
			fmt.Println("close failed", err)
		}
	}()

	//c.SetLinger(0)

	host, port, err = getOriginalDst(rawConn)

	needProxied = filters.CheckTargetIP(host)

	if !needProxied {
		host, needProxied, buf, err = filters.ExecFilters(downstreamReader)
		if err != nil {
			// do something
			fmt.Println(err)
			return
		}
	}

	if needProxied {
		d = proxyDialer
	} else {
		d = proxy.Direct
	}

	sendConn, err := conn.DialUpstream(d, "tcp", fmt.Sprintf("%s:%d", host, port), timeout)

	if err != nil {
		fmt.Println(err)
		return
	}

	if buf != nil {
		buf.Write(sendConn)
	}

	defer sendConn.CloseWrite()

	go copyData(c, sendConn)

	io.Copy(sendConn, downstreamReader)
	sendConn.CloseRead()
}

func getsockopt(s int, level int, name int, val uintptr, vallen *uint32) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(name), uintptr(val), uintptr(unsafe.Pointer(vallen)), 0)
	if e1 != 0 {
		err = e1
	}
	return
}

//func copyData(wg *sync.WaitGroup, conn1, conn2 net.Conn) {
func copyData(dst, src *conn.IdleTimeoutConn) {

	defer dst.CloseWrite()
	defer src.CloseRead()

	r, w := io.Pipe()
	go func() {
		defer w.Close()
		if _, err := io.Copy(w, src); err != nil {
			return
		}
	}()
	defer r.Close()
	if _, err := io.Copy(dst, r); err != nil {
		return
	}
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
		fmt.Printf("File failed: %s", err.Error())
		return "", 0, err
	}
	defer clientConnFile.Close()

	//sa, err := syscall.GetsockoptInet4Addr(int(clientConnFile.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)

	var sa syscall.RawSockaddrInet4
	var size = uint32(unsafe.Sizeof(sa))

	err = getsockopt(int(clientConnFile.Fd()), syscall.IPPROTO_IP, 80, uintptr(unsafe.Pointer(&sa)), &size)
	if err != nil {
		fmt.Printf("GETORIGINALDST failed: %s ", err.Error())
		return "", 0, err
	}

	addr := net.IPv4(sa.Addr[0], sa.Addr[1], sa.Addr[2], sa.Addr[3])
	nport := sa.Port
	lport := binary.BigEndian.Uint16((*(*[2]byte)(unsafe.Pointer(&nport)))[:])
	port := uint16(lport)

	return addr.String(), port, nil

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
