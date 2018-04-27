package filter

import (
	"bufio"
	"fmt"
	"github.com/joycn/puckgo/datasource"
	"io"
	"net"
	"strings"
)

var (
	filterMap = make(map[string]*Filter)
)

// Buffer write buffer for net.Conn
type Buffer interface {
	Write(io.Writer) error
}

// Filter Filter used to check request with name
type Filter struct {
	Func func(r *bufio.Reader) (string, Buffer, error)
	Name string
}

// Filters Filter list witch match conditions used to check request
type Filters struct {
	m  map[int]*Filter
	al *datasource.AccessList
}

// NewFilters create a new filters with accesslist
func NewFilters(al *datasource.AccessList) *Filters {
	filters := &Filters{al: al}
	filters.m = make(map[int]*Filter)
	return filters
}

// AddFilter add a new filter into filters
func (filters *Filters) AddFilter(name string, port int) error {
	//n := f.Name
	if _, ok := filters.m[port]; !ok {
		if f, ok := filterMap[strings.ToLower(name)]; ok {
			filters.m[port] = f
			return nil
		}
		return fmt.Errorf("%s filter not found", name)
	}
	return fmt.Errorf("filter exist")
}

// RemoveFilter remove a new filters from filters
func (filters *Filters) RemoveFilter(port int) error {
	//n := f.Name
	if _, ok := filters.m[port]; ok {
		delete(filters.m, port)
		return nil
	}
	return fmt.Errorf("filter not exist")
}

// CheckTargetIP check dst ip need to proxied
func (filters *Filters) CheckTargetIP(target string) bool {
	ip := net.ParseIP(target)
	al := filters.al
	return al.MatchIP(ip)
}

// ExecFilters exec all filter to check request
func (filters *Filters) ExecFilters(r *bufio.Reader, port int) (string, Buffer, error) {
	var (
		host string
		buf  Buffer
		err  error
	)

	if f, ok := filters.m[port]; ok {
		host, buf, err = f.Func(r)
		return host, buf, err
	}
	return host, buf, err
}

// Match check host whether should be proxied
func (filters *Filters) Match(host string) bool {
	ip := net.ParseIP(host)
	al := filters.al
	if ip != nil {
		return al.MatchIP(ip)
	}
	return al.MatchDomain(host)

}
