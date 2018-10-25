package dnsforward

import (
	"bufio"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/datasource"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"strings"
	"sync"
)

type request struct {
	Name   string
	Remote *net.UDPAddr
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

func installIPset(msg *dns.Msg) {
	for _, r := range msg.Answer {
		h := r.Header()
		ttl := h.Ttl
		if h.Rrtype == dns.TypeA {
			a := r.(*dns.A)
			netlink.IPsetUpdateTimeout("vpn", a.A, ttl)
		}
	}
}

func readFromServer(r, s *net.UDPConn, ipset bool) {
	b := make([]byte, 1500)
	for {
		n, err := r.Read(b)
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

// StartDNS start dns server to forward or answer dns query
func StartDNS(al *datasource.AccessList, dnsConfig *config.DNSConfig) error {
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

	addr, err := net.ResolveUDPAddr("udp", dnsConfig.Listen)
	if err != nil {
		logrus.Error(err)
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logrus.Error(err)
		return err
	}

	go readFromServer(defaultServerConn, conn, false)
	go readFromServer(specifiedServerConn, conn, true)

	for {
		b := make([]byte, 1500)
		n, remote, err := conn.ReadFromUDP(b)
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

			if matched {
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
