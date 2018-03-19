package proxy

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"os"
	"socks5/conn"
	"socks5/sni"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var dialer proxy.Dialer
var isProxy bool

func StartProxy(listen, upstream string, timeout time.Duration) {
	if len(os.Args) > 2 && os.Args[2] == "proxy" {
		isProxy = true
	}
	dialer, _ = proxy.SOCKS5("tcp", upstream, nil, proxy.Direct)
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
		go handleConn(conn, dialer, timeout)
	}
}

func handleConn(rawConn *net.TCPConn, d proxy.Dialer, timeout time.Duration) {

	c := conn.NewIdleTimeoutConn(rawConn, timeout)

	downstreamReader := bufio.NewReader(c)

	//c.SetDeadline(time.Now().Add(time.Duration(3) * time.Second))
	target, err := getOriginalDst(rawConn)
	if isProxy {
		d = &net.Dialer{}
	}
	//sendConn, err := d.Dial("tcp", target)

	sendConn, err := conn.DialUpstream(d, "tcp", target, timeout)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer sendConn.Close()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go copyData(c, sendConn, wg)

	io.Copy(sendConn, downstreamReader)

	wg.Wait()
	fmt.Println("finished")
}

func getsockopt(s int, level int, name int, val uintptr, vallen *uint32) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(name), uintptr(val), uintptr(unsafe.Pointer(vallen)), 0)
	if e1 != 0 {
		err = e1
	}
	return
}

//func copyData(wg *sync.WaitGroup, conn1, conn2 net.Conn) {
func copyData(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	fmt.Println("copyData start", time.Now())

	r, w := io.Pipe()
	defer func() {
		fmt.Println("copyData finished", time.Now())
		wg.Done()
	}()

	go func() {
		defer w.Close()
		if _, err := io.Copy(w, src); err != nil {
			fmt.Println("src", err)
			return
		} else {
			fmt.Println("src finished")
		}
	}()
	defer r.Close()
	if _, err := io.Copy(dst, r); err != nil {
		fmt.Println("dst", err)
		return
	} else {
		fmt.Println("dst finished")
	}
	//for {
	////src.SetWriteDeadline(time.Now().Add(time.Second * 3))
	////dst.SetReadDeadline(time.Now().Add(time.Second * 3))
	//if n, err := src.Read(b); err != nil {
	//fmt.Println("copy data", err, "write", n)
	//break
	//} else {
	//dst.Write(b[:n])
	//}
	////if n, err := io.Copy(conn1, conn2); err != nil {
	////if e := conn1.Close(); e != nil {
	////fmt.Println("close failed", e)
	////}
	////fmt.Println("copy data", err, "write", n)
	////break
	////} else {
	////fmt.Println("write", n)
	////}

	//}
}

func isHTTPRequest(r *bufio.Reader) bool {

	firstChar, err := r.Peek(1)

	if err != nil {
		return false
	}

	ch := firstChar[0]

	if (ch < 'A' || ch > 'Z') && ch != '_' && ch != '-' {
		return false
	}

	return true
}

func handleHTTP(r *bufio.Reader) (*http.Request, error) {

	fmt.Println("http", time.Now())

	req, err := http.ReadRequest(r)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fmt.Println(req.Host)
	return req, nil
}

func getOriginalDst(clientConn *net.TCPConn) (string, error) {
	clientConnFile, err := clientConn.File()
	if err != nil {
		fmt.Printf("File failed: %s", err.Error())
		return "", err
	}

	//sa, err := syscall.GetsockoptInet4Addr(int(clientConnFile.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)

	var sa syscall.RawSockaddrInet4
	var size = uint32(unsafe.Sizeof(sa))

	err = getsockopt(int(clientConnFile.Fd()), syscall.IPPROTO_IP, 80, uintptr(unsafe.Pointer(&sa)), &size)
	if err != nil {
		fmt.Printf("GETORIGINALDST failed: %s ", err.Error())
		return "", err
	}

	addr := net.IPv4(sa.Addr[0], sa.Addr[1], sa.Addr[2], sa.Addr[3]).String()
	nport := sa.Port
	lport := binary.BigEndian.Uint16((*(*[2]byte)(unsafe.Pointer(&nport)))[:])
	port := strconv.Itoa(int(lport))

	return addr + ":" + port, nil
}

func handleHTTPS(b *bufio.Reader) (io.Reader, error) {

	fmt.Println("https", time.Now())

	_, err := sni.ReadServerName(b)
	if err != nil {
		return b, nil
	} else {
		return nil, err
	}
}
