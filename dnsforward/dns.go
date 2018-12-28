package dnsforward

import (
	"bufio"
	"fmt"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/datasource"
	"github.com/joycn/ttlcache"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

// dnsCache cache ip for domain
var dnsCache *ttlcache.Cache

// socksCache used for map ip to domain
var socksCache *ttlcache.Cache

// GetDomain search ip in dnsCache and return related domain
func GetDomain(ip string) string {
	record, ok := socksCache.Load(ip)
	if !ok {
		return ip
	}

	return record.(string)
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

func installIPset(msg *dns.Msg) {
	for _, r := range msg.Answer {
		h := r.Header()
		ttl := h.Ttl
		switch h.Rrtype {
		case dns.TypeA:
			a := r.(*dns.A)
			netlink.IPsetUpdateTimeout("vpn", a.A, ttl)
			host := updateCache(socksCache, a.A.String(), h.Name, time.Duration(ttl)*time.Second, false)
			dnsCache.Store(host, a.A, time.Duration(ttl)*time.Second, false)
		case dns.TypeCNAME:
			cname := r.(*dns.CNAME)
			//netlink.IPsetUpdateTimeout("vpn", a.A, ttl)
			updateCache(socksCache, cname.Target, h.Name, time.Duration(ttl)*time.Second, false)
		}
	}
}

func readFromServer(r, s net.PacketConn, ipset bool) {
	b := make([]byte, 1500)
	for {
		n, _, err := r.ReadFrom(b)
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

		m.Lock()
		rq, ok := onFlyMap[msg.Id]
		m.Unlock()
		if !ok {
			continue
		}
		msg = new(dns.Msg)
		err = msg.Unpack(b[:n])
		if err == nil {
			if ipset {
				installIPset(msg)
			}
			s.WriteTo(b[:n], rq.Remote)
		}
	}
}

var onFlyMap map[uint16]*request
var targetConn map[datasource.MatchAction]*net.UDPConn
var m sync.Mutex

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

func replyFromCache(query *dns.Msg) *dns.Msg {
	if len(query.Question) == 0 {
		return nil
	}

	if query.Question[0].Qtype != dns.TypeA {
		return nil
	}

	domain := query.Question[0].Name
	r, found := dnsCache.Load(domain)
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
	fmt.Println(reply)
	return reply
}

// StartDNS start dns server to forward or answer dns query
func StartDNS(al *datasource.AccessList, dnsConfig *config.DNSConfig) error {
	dnsCache = ttlcache.New(time.Minute, 5*time.Minute)
	socksCache = ttlcache.New(time.Minute, 5*time.Minute)
	onFlyMap = make(map[uint16]*request)
	targetConn = make(map[datasource.MatchAction]*net.UDPConn)
	m = sync.Mutex{}

	defaultServerAddr, err := net.ResolveUDPAddr("udp", dnsConfig.DefaultServer)
	defaultServerConn, err := net.DialUDP("udp", nil, defaultServerAddr)
	if err != nil {
		logrus.Error(err)
		return err
	}

	specifiedServerAddr, err := net.ResolveUDPAddr("udp", dnsConfig.SpecifiedServer)
	specifiedServerConn, err := net.DialUDP("udp", nil, specifiedServerAddr)
	if err != nil {
		logrus.Error(err)
		return err
	}

	conn, err := listenUDP("udp", dnsConfig.Listen)
	if err != nil {
		logrus.Error(err)
		return err
	}
	go readFromServer(defaultServerConn, conn, false)
	go readFromServer(specifiedServerConn, conn, true)

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

		if reply := replyFromCache(msg); reply != nil {
			data, err := reply.Pack()
			if err == nil {
				conn.WriteTo(data, remote)
				continue
			}
		}

		if len(msg.Question) > 0 {
			q := msg.Question[0]
			name := q.Name
			matched := al.MatchDomain(name)
			m.Lock()
			onFlyMap[msg.Id] = &request{
				Name:   msg.Question[0].Name,
				Remote: remote,
			}
			m.Unlock()

			if matched && q.Qtype == dns.TypeA {
				_, err = specifiedServerConn.Write(b[:n])
			} else {
				_, err = defaultServerConn.Write(b[:n])
			}
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}

}
