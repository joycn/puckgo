package conn

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"github.com/joycn/datasource"
	"github.com/joycn/socks"
	"net"
)

const magicChar = "RojvQ_OWDeEGMBXIZF4Cy5nVJgqiSs3-1twbHKNf+rT8Ldm2ckPhl79zAxauYp6U"
const paddingRune = '0'

// CryptoDialer dial target and return a cipher conn
type CryptoDialer struct {
	Network      string
	UpstreamAddr string
	key          []byte
	match        bool
	*datasource.AccessList
}

// NewCryptoDialer create CryptoDialer with a base64 string
func NewCryptoDialer(network, upstream, key string, match bool, ma *datasource.AccessList) (*CryptoDialer, error) {
	var ngEncoder = base64.NewEncoding(magicChar).WithPadding(paddingRune)
	data, err := ngEncoder.DecodeString(key)
	if err != nil {
		return nil, err
	}
	return &CryptoDialer{Network: network, UpstreamAddr: upstream, key: data, AccessList: ma, match: match}, nil
}

// CryptoConn conn with cipher
type CryptoConn struct {
	*net.TCPConn
	r *cipher.StreamReader
	w *cipher.StreamWriter
}

func (c *CryptoConn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *CryptoConn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

// Dial target and return a cipher conn
func (d *CryptoDialer) Dial(addr *socks.AddrSpec) (c net.Conn, err error) {
	if d.match {
		if match := d.MatchDomain(addr.FQDN); !match {
			return nil, fmt.Errorf("FQDN not match")
		}
	}

	if c, err = net.Dial(d.Network, d.UpstreamAddr); err != nil {
		return nil, err
	}
	cc := &CryptoConn{TCPConn: c.(*net.TCPConn)}
	//defer func() {
	//if err != nil {
	//c.Close()
	//}
	//}()
	block, err := aes.NewCipher(d.key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize)
	iv := ciphertext[:aes.BlockSize]
	stream := cipher.NewOFB(block, iv[:])
	cc.r = &cipher.StreamReader{S: stream, R: c}
	cc.w = &cipher.StreamWriter{S: stream, W: c}
	if err = socks.Connect(cc, addr, true); err != nil {
		return nil, err
	}

	return cc, nil
}

// newCryptoConn use key to create a CryptoConn for c
func newCryptoConn(c net.Conn, key []byte) (*CryptoConn, error) {
	s, ok := c.(*net.TCPConn)
	if !ok {
		fmt.Errorf("must be tcp conn")
	}
	cc := &CryptoConn{TCPConn: s}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize)
	iv := ciphertext[:aes.BlockSize]
	stream := cipher.NewOFB(block, iv[:])
	cc.r = &cipher.StreamReader{S: stream, R: c}
	cc.w = &cipher.StreamWriter{S: stream, W: c}
	return cc, nil
}

// CryptoReception get target ip and port and return a CryptoConn
type CryptoReception struct {
	s   *socks.Server
	key []byte
}

// NewCryptoReception return a CryptoReception with noauth socks server
func NewCryptoReception(s *socks.Server, key string) (*CryptoReception, error) {
	var ngEncoder = base64.NewEncoding(magicChar).WithPadding(paddingRune)
	data, err := ngEncoder.DecodeString(key)
	if err != nil {
		return nil, err
	}
	r := &CryptoReception{s: s, key: data}
	r.s.NoAuth = true
	return r, nil
}

// Recept get target ip and port and return a CryptoConn
func (cr *CryptoReception) Recept(c net.Conn) (*socks.AddrSpec, net.Conn, error) {
	cc, err := newCryptoConn(c, cr.key)
	if err != nil {
		return nil, nil, err
	}
	addr, _, err := cr.s.Recept(cc)
	if err != nil {
		return nil, nil, err
	}
	return addr, cc, nil
}
