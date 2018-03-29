package datasource

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// MatchAction action for matched url
type MatchAction bool

const (
	// Except forward to exceptive server
	Except MatchAction = true
	// Default forward to default server
	Default MatchAction = false
)

// DomainMap map info for dns name and actions
type DomainMap map[string]bool

// AccessList dns or ip subnets for proxy
type AccessList struct {
	Domains DomainMap
	Subnets []*net.IPNet
	*sync.RWMutex
}

type accessListConfig struct {
	Domains []string `yaml:"domains"`
	Subnets []string `yaml:"subnets"`
}

func newAccessList(alc *accessListConfig) (*AccessList, error) {
	al := new(AccessList)
	al.Domains = make(DomainMap)
	al.RWMutex = new(sync.RWMutex)

	for _, domain := range alc.Domains {
		al.Domains[domain] = true
	}

	for _, subnet := range alc.Subnets {
		if _, cidr, err := net.ParseCIDR(subnet); err == nil {
			al.Subnets = append(al.Subnets, cidr)
		} else {
			return nil, err
		}
	}
	return al, nil
}

// AddDomain add a domain to al
func (al *AccessList) AddDomain(domain string) {
	al.Lock()
	defer al.Unlock()
	al.Domains[domain] = true
}

// AddSubnet add subnet to al
func (al *AccessList) AddSubnet(subnet string) error {
	al.Lock()
	defer al.Unlock()
	if _, cidr, err := net.ParseCIDR(subnet); err == nil {
		al.Subnets = append(al.Subnets, cidr)
	} else {
		return err
	}
	return nil
}

// GetAccessList get access list from source
func GetAccessList(source string) (*AccessList, error) {
	tokens := strings.SplitN(source, ":", 2)
	if len(tokens) != 2 {
		return nil, fmt.Errorf("datasource format error")
	}

	switch tokens[0] {
	case "file":
		return AccessListFromFile(tokens[1])
	case "etcd":
	}
	return nil, fmt.Errorf("unsupported datasource")
}

// MatchDomain check whether name is in matchactions
func (al *AccessList) MatchDomain(name string) bool {
	dm := al.Domains
	al.RLock()
	defer al.RUnlock()
	name = strings.TrimSuffix(name, ".")
	var tokens []string
	for {
		if _, ok := dm[name]; ok {
			return true
		}
		tokens = strings.SplitN(name, ".", 2)
		if len(tokens) == 1 {
			break
		}
		name = tokens[1]
	}
	return false
}

// MatchIP check whether name is in matchactions
func (al *AccessList) MatchIP(target net.IP) bool {
	sn := al.Subnets
	al.RLock()
	defer al.RUnlock()
	for _, subnet := range sn {
		if subnet.Contains(target) {
			return true
		}
	}
	return false
}
