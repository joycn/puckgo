package conn

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/joycn/socks"
	"github.com/sirupsen/logrus"
)

const maxHeaderSize = 8192

// prefixConn wraps a *net.TCPConn with prefix bytes prepended to reads.
// It embeds *net.TCPConn so CloseWrite() is inherited, satisfying StreamConn.
type prefixConn struct {
	*net.TCPConn
	reader io.Reader
}

func newPrefixConn(prefix []byte, c *net.TCPConn) *prefixConn {
	return &prefixConn{
		TCPConn: c,
		reader:  io.MultiReader(bytes.NewReader(prefix), c),
	}
}

func (pc *prefixConn) Read(b []byte) (int, error) {
	return pc.reader.Read(b)
}

// MultiReception detects the protocol on an incoming connection and
// delegates to the appropriate handler (SOCKS5 or HTTP/HTTPS proxy).
type MultiReception struct {
	socks *socks.Server
}

// NewMultiReception creates a MultiReception wrapping the given socks.Server.
func NewMultiReception(s *socks.Server) *MultiReception {
	return &MultiReception{socks: s}
}

// Recept implements conn.Reception. It peeks at the first byte to detect
// the protocol and dispatches accordingly.
func (m *MultiReception) Recept(c net.Conn) (*socks.AddrSpec, net.Conn, error) {
	var peek [1]byte
	if _, err := io.ReadFull(c, peek[:]); err != nil {
		return nil, nil, fmt.Errorf("failed to read first byte: %w", err)
	}

	if peek[0] == 0x05 {
		// SOCKS5 protocol
		tcpConn, ok := c.(*net.TCPConn)
		if !ok {
			return nil, nil, fmt.Errorf("expected *net.TCPConn, got %T", c)
		}
		pc := newPrefixConn(peek[:], tcpConn)
		return m.socks.Recept(pc)
	}

	// HTTP protocol — read the full headers
	tcpConn, ok := c.(*net.TCPConn)
	if !ok {
		return nil, nil, fmt.Errorf("expected *net.TCPConn, got %T", c)
	}

	headers, overflow, err := readHeaders(c, peek[0])
	if err != nil {
		return nil, nil, err
	}

	// Parse request line (first line)
	headerStr := string(headers)
	lineEnd := strings.Index(headerStr, "\r\n")
	if lineEnd < 0 {
		return nil, nil, fmt.Errorf("malformed HTTP request: no CRLF in headers")
	}
	method, uri, _, err := parseRequestLine(headerStr[:lineEnd])
	if err != nil {
		return nil, nil, err
	}

	logrus.WithFields(logrus.Fields{
		"method": method,
		"uri":    uri,
	}).Debug("HTTP proxy request")

	if method == "CONNECT" {
		return m.handleHTTPConnect(uri, tcpConn)
	}
	return m.handleHTTPProxy(method, uri, headers, overflow, tcpConn)
}

// handleHTTPConnect handles CONNECT host:port HTTP/1.1 requests.
func (m *MultiReception) handleHTTPConnect(uri string, c *net.TCPConn) (*socks.AddrSpec, net.Conn, error) {
	host, portStr, err := net.SplitHostPort(uri)
	if err != nil {
		// No port specified, default to 443
		host = uri
		portStr = "443"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid port in CONNECT URI %q: %w", uri, err)
	}

	addr := &socks.AddrSpec{
		FQDN: host,
		Port: port,
	}

	// Send 200 Connection Established
	_, err = c.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send CONNECT response: %w", err)
	}

	return addr, c, nil
}

// handleHTTPProxy handles plain HTTP proxy requests (GET http://host/path HTTP/1.1).
func (m *MultiReception) handleHTTPProxy(method, uri string, headers, overflow []byte, c *net.TCPConn) (*socks.AddrSpec, net.Conn, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid HTTP proxy URI %q: %w", uri, err)
	}

	host := u.Hostname()
	if host == "" {
		return nil, nil, fmt.Errorf("no host in HTTP proxy URI %q", uri)
	}

	portStr := u.Port()
	if portStr == "" {
		portStr = "80"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid port in HTTP proxy URI %q: %w", uri, err)
	}

	addr := &socks.AddrSpec{
		FQDN: host,
		Port: port,
	}

	// Rewrite request: absolute URI -> relative path
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	// Build rewritten request
	rewritten := rewriteHTTPRequest(method, path, host, headers)

	// Combine rewritten headers with any body overflow
	var prefix []byte
	if len(overflow) > 0 {
		prefix = make([]byte, len(rewritten)+len(overflow))
		copy(prefix, rewritten)
		copy(prefix[len(rewritten):], overflow)
	} else {
		prefix = rewritten
	}

	pc := newPrefixConn(prefix, c)
	return addr, pc, nil
}

// rewriteHTTPRequest rewrites the HTTP request headers:
// - Replaces absolute URI with relative path
// - Injects Connection: close
// - Removes Proxy-Connection and Proxy-Authorization headers
func rewriteHTTPRequest(method, path, host string, rawHeaders []byte) []byte {
	headerStr := string(rawHeaders)

	// Find end of first line
	lineEnd := strings.Index(headerStr, "\r\n")
	if lineEnd < 0 {
		return rawHeaders
	}
	remainingHeaders := headerStr[lineEnd+2:] // skip first line + CRLF

	var buf bytes.Buffer
	// Write new request line
	buf.WriteString(method)
	buf.WriteByte(' ')
	buf.WriteString(path)
	buf.WriteString(" HTTP/1.1\r\n")

	// Process remaining headers
	hasHost := false
	hasConnection := false
	lines := strings.Split(remainingHeaders, "\r\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		// Skip proxy-specific headers
		if strings.HasPrefix(lower, "proxy-connection:") ||
			strings.HasPrefix(lower, "proxy-authorization:") {
			continue
		}
		// Replace Connection header
		if strings.HasPrefix(lower, "connection:") {
			hasConnection = true
			buf.WriteString("Connection: close\r\n")
			continue
		}
		if strings.HasPrefix(lower, "host:") {
			hasHost = true
		}
		buf.WriteString(line)
		buf.WriteString("\r\n")
	}

	if !hasHost {
		buf.WriteString("Host: ")
		buf.WriteString(host)
		buf.WriteString("\r\n")
	}
	if !hasConnection {
		buf.WriteString("Connection: close\r\n")
	}

	// End of headers
	buf.WriteString("\r\n")
	return buf.Bytes()
}

// readHeaders reads HTTP headers from the connection, starting with the
// already-peeked first byte. Returns the header bytes (up to and including
// \r\n\r\n) and any overflow body bytes that were read past the delimiter.
func readHeaders(c net.Conn, firstByte byte) (headers, overflow []byte, err error) {
	buf := make([]byte, 1, maxHeaderSize)
	buf[0] = firstByte

	tmp := make([]byte, 4096)
	for {
		n, err := c.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read HTTP headers: %w", err)
		}

		// Check for end of headers
		if idx := bytes.Index(buf, []byte("\r\n\r\n")); idx >= 0 {
			endOfHeaders := idx + 4
			return buf[:endOfHeaders], buf[endOfHeaders:], nil
		}

		if len(buf) > maxHeaderSize {
			return nil, nil, fmt.Errorf("HTTP headers exceed maximum size (%d bytes)", maxHeaderSize)
		}
	}
}

// parseRequestLine parses "METHOD URI PROTO" from the first line.
func parseRequestLine(line string) (method, uri, proto string, err error) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("malformed HTTP request line: %q", line)
	}
	return parts[0], parts[1], parts[2], nil
}
