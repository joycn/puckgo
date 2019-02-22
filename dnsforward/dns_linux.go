package dnsforward

import (
	"bufio"
	"context"
	"fmt"
	"github.com/joycn/datasource"
	"github.com/joycn/puckgo/network"
	"github.com/joycn/socks"
	"github.com/joycn/ttlcache"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

const (
	defaultTTL = 3600
)

// DNSConfig dns config params for dns server
type DNSConfig struct {
	Listen          string
	DefaultServer   string
	SpecifiedServer string
}

// DNSForwarder forward target dns request to special server
type DNSForwarder struct {
	listen     string
	al         *datasource.AccessList
	dnsCache   *ttlcache.Cache
	socksCache *ttlcache.Cache
	resolver   *net.Resolver
}

type resolverDial func(ctx context.Context, network, address string) (net.Conn, error)

// NewDNSForwarder return a new dns forwarder
func NewDNSForwarder(listen, upstream string, al *datasource.AccessList, dialer network.Dialer) (*DNSForwarder, error) {
	addr, err := net.ResolveTCPAddr("tcp", upstream)
	if err != nil {
		return nil, err
	}
	f := &DNSForwarder{listen: listen, al: al}
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		upstreamAddr := &socks.AddrSpec{IP: addr.IP, FQDN: addr.IP.To4().String(), Port: addr.Port}
		return dialer.Dial(upstreamAddr)
	}
	f.resolver = &net.Resolver{PreferGo: true, Dial: dial}
	f.dnsCache = ttlcache.New(time.Minute, 5*time.Minute)
	f.socksCache = ttlcache.New(time.Minute, 5*time.Minute)
	return f, nil
}

//// Dial speicial dial to puckgo server
//func (f *DNSForwarder) Dial(ctx context.Context, network, address string) (Conn, error) {
//addr := &socks.AddrSpec{IP: net.ParseIP(upstream)}
//}

// GetDomain search ip in dnsCache and return related domain
func (f *DNSForwarder) GetDomain(ip string) (string, bool) {
	record, ok := f.socksCache.Load(ip)
	if !ok {
		return ip, ok
	}

	return record.(string), ok
}

type request struct {
	Name   string
	Remote net.Addr
}

func excludedDNS(filename string) (map[string]bool, error) {
	ret := make(map[string]bool)
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(f)

	for {
		url, err := reader.ReadString('\n')
		url = strings.TrimSpace(url)
		if err != nil {
			break
		}
		ret[url] = true
	}
	return ret, nil
}

func updateCache(c *ttlcache.Cache, k, v string, timeout time.Duration, persistent bool) string {
	final := v
	last, found := c.Load(v)
	if found {
		final = last.(string)
		//c.Delete(v)
	}
	c.Store(k, final, timeout, persistent)
	return final
}

func (f *DNSForwarder) installIPset(msg *dns.Msg) {
	for _, r := range msg.Answer {
		h := r.Header()
		ttl := h.Ttl
		switch h.Rrtype {
		case dns.TypeA:
			a := r.(*dns.A)
			netlink.IPsetUpdateTimeout("vpn", a.A, ttl)
			host := updateCache(f.socksCache, a.A.String(), h.Name, time.Duration(ttl)*time.Second, false)
			f.dnsCache.Store(host, a.A, time.Duration(ttl)*time.Second, false)
		case dns.TypeCNAME:
			cname := r.(*dns.CNAME)
			//netlink.IPsetUpdateTimeout("vpn", a.A, ttl)
			updateCache(f.socksCache, cname.Target, h.Name, time.Duration(ttl)*time.Second, false)
		}
	}
}

func (f *DNSForwarder) updateIPset(name string, result []net.IPAddr) {
	for _, r := range result {
		if r.IP.To4() != nil {
			logrus.Debug(fmt.Sprintf("update ipset vpn %s", r.IP.String()))
			netlink.IPsetUpdateTimeout("vpn", r.IP, defaultTTL)
			updateCache(f.socksCache, r.IP.String(), name, defaultTTL*time.Second, false)
		}
		//f.dnsCache.Store(host, a.A, time.Duration(ttl)*time.Second, false)
	}
}

func listenUDP(network, address string) (c net.PacketConn, err error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}

	if err = syscall.SetsockoptInt(s, syscall.SOL_IP, syscall.IP_TRANSPARENT, 1); err != nil {
		return nil, err
	}
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}
	lsa := syscall.SockaddrInet4{Port: addr.Port}
	copy(lsa.Addr[:], addr.IP.To4())
	if err = syscall.Bind(s, &lsa); err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(s), "")
	defer f.Close()
	return net.FilePacketConn(f)
}

func reply(query *dns.Msg, result []net.IPAddr) *dns.Msg {
	if len(query.Question) == 0 {
		return nil
	}

	if query.Question[0].Qtype != dns.TypeA {
		return nil
	}

	domain := query.Question[0].Name

	reply := &dns.Msg{}
	reply.SetReply(query)
	reply.Authoritative = true
	for _, r := range result {
		if r.IP.To4() == nil {
			continue
		}
		aRecord := &dns.A{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    defaultTTL,
			},
			A: r.IP,
		}
		reply.Answer = append(reply.Answer, aRecord)
	}
	return reply
}

func (f *DNSForwarder) replyFromCache(query *dns.Msg) *dns.Msg {
	if len(query.Question) == 0 {
		return nil
	}

	if query.Question[0].Qtype != dns.TypeA {
		return nil
	}

	domain := query.Question[0].Name
	r, found := f.dnsCache.Load(domain)
	if !found {
		return nil
	}

	reply := &dns.Msg{}
	reply.SetReply(query)
	reply.Authoritative = true
	aRecord := &dns.A{
		Hdr: dns.RR_Header{
			Name:   domain,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		A: r.(net.IP),
	}
	reply.Answer = append(reply.Answer, aRecord)
	return reply
}

func (f *DNSForwarder) process(conn net.PacketConn, raddr, saddr net.Addr, msg *dns.Msg) {
	ctx := context.Background()
	q := msg.Question[0]
	name := q.Name
	matched := f.al.MatchDomain(name)
	var result = []net.IPAddr{}
	var err error
	if !matched || q.Qtype != dns.TypeA {
		c, err := net.DialUDP("udp", nil, saddr.(*net.UDPAddr))
		if err != nil {
			logrus.Error(err)
			return
		}
		data, err := msg.Pack()
		if err != nil {
			logrus.Error(err)
			return
		}
		c.SetDeadline(time.Now().Add(time.Second * 3))
		if _, err := c.Write(data); err != nil {
			logrus.Error(err)
			return
		}

		b := make([]byte, 1500)
		if n, err := c.Read(b); err == nil {
			conn.WriteTo(b[:n], raddr)
		} else {
			logrus.Error(err)
			return
		}
	}
	result, err = f.resolver.LookupIPAddr(ctx, name)

	if err != nil {
		logrus.Error(err)
		return
	}

	f.updateIPset(name, result)
	replyMsg := reply(msg, result)

	if replyMsg == nil {
		return
	}

	data, err := replyMsg.Pack()
	if err != nil {
		logrus.Error(err)
		return
	}
	conn.WriteTo(data, raddr)
	return
}

// StartDNS start dns server to forward or answer dns query
func (f *DNSForwarder) StartDNS() error {

	var conn net.PacketConn

	resolverAddr, err := net.ResolveUDPAddr("udp", f.listen)
	if err != nil {
		logrus.Error(err)
		return err
	}

	conn, err = listenUDP("udp", f.listen)
	if err != nil {
	}

	for {
		b := make([]byte, 1500)
		n, remote, err := conn.ReadFrom(b)
		if err != nil {
			logrus.Error(err)
			continue
		}
		msg := new(dns.Msg)
		err = msg.Unpack(b[:n])
		if err != nil {
			logrus.Error(err)
			continue
		}
		if len(msg.Question) == 0 {
			logrus.Error(fmt.Errorf("no question found"))
			continue
		}

		go f.process(conn, remote, resolverAddr, msg)

	}
}
