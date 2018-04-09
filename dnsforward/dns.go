package dnsforward

import (
	"bufio"
	"github.com/joycn/puckgo/config"
	"github.com/joycn/puckgo/datasource"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
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

func readFromServer(r, s *net.UDPConn) {
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
			s.WriteTo(b[:n], rq.Remote)
		}
	}
}

var onFlyMap map[uint16]*request
var targetConn map[datasource.MatchAction]*net.UDPConn
var m sync.Mutex

// StartDNS start dns server to forward or answer dns query
func StartDNS(al *datasource.AccessList, proxyMatch bool, dnsConfig *config.DNSConfig) error {
	//ma, err := datasource.GetMatchActions(source)
	//if err != nil {
	//logrus.Error(err)
	//return err
	//}
	onFlyMap = make(map[uint16]*request)
	targetConn = make(map[datasource.MatchAction]*net.UDPConn)
	m = sync.Mutex{}

	defaultServerAddr, err := net.ResolveUDPAddr("udp", dnsConfig.DefaultServer)
	defaultServerConn, err := net.DialUDP("udp", nil, defaultServerAddr)
	if err != nil {
		logrus.Error(err)
		return err
	}
	//targetConn[datasource.Default] = defaultServerConn

	exceptiveServerAddr, err := net.ResolveUDPAddr("udp", dnsConfig.ExceptiveServer)
	exceptiveServerConn, err := net.DialUDP("udp", nil, exceptiveServerAddr)
	if err != nil {
		logrus.Error(err)
		return err
	}
	//targetConn[datasource.Except] = exceptiveServerConn

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

	//for _, s := range targetConn {
	go readFromServer(defaultServerConn, conn)
	go readFromServer(exceptiveServerConn, conn)
	//}

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
			if matched == proxyMatch && !config.PublicService {
				if q.Qtype == 1 && q.Qclass == 1 && !remote.IP.IsLoopback() {
					msg.Response = true
					a := &dns.A{Hdr: dns.RR_Header{Name: name, Rrtype: 1, Class: 1, Ttl: 3600}, A: net.IPv4(111, 111, 111, 111)}
					msg.Answer = append(msg.Answer, a)
					b, err := msg.Pack()
					if err == nil {
						conn.WriteTo(b, remote)
						continue
					}
				}
			}
			m.Lock()
			onFlyMap[msg.Id] = &request{
				Name:   msg.Question[0].Name,
				Remote: remote,
			}
			m.Unlock()

			if matched {
				_, err = exceptiveServerConn.Write(b[:n])
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
