package filter

import (
	"bufio"
	"fmt"
	"github.com/joycn/puckgo/datasource"
	"io"
	"net"
)

// Action action for request
type Action int

// Buffer write buffer for net.Conn
type Buffer interface {
	Write(io.Writer) error
}

const (
	// Continue exec next filter
	Continue Action = iota
	// Stop stop exec filters
	Stop
	// Again exec current filter again
	Again
)

// CheckFunc filter function to check request
type CheckFunc func(r *bufio.Reader) (string, Action, Buffer, error)

// Filter Filter used to check request with name
type Filter struct {
	Func func(r *bufio.Reader) (string, Action, Buffer, error)
	Name string
}

// Filters Filter list witch match conditions used to check request
type Filters struct {
	m  map[string]*Filter
	al *datasource.AccessList
}

// NewFilters create a new filters with accesslist
func NewFilters(al *datasource.AccessList) *Filters {
	filters := &Filters{al: al}
	filters.m = make(map[string]*Filter)
	return filters
}

// AddFilter add a new filter into filters
func (filters *Filters) AddFilter(f *Filter) error {
	n := f.Name
	if _, ok := filters.m[n]; !ok {
		filters.m[n] = f
		return nil
	}
	return fmt.Errorf("filter exist")
}

// RemoveFilter remove a new filters from filters
func (filters *Filters) RemoveFilter(f *Filter) error {
	n := f.Name
	if _, ok := filters.m[n]; ok {
		delete(filters.m, n)
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
func (filters *Filters) ExecFilters(r *bufio.Reader) (string, bool, Buffer, error) {
	var (
		host   string
		action Action
		buf    Buffer
		err    error
	)

	for _, f := range filters.m {
		for {
			host, action, buf, err = f.Func(r)
			if action != Again {
				break
			}
		}

		if action == Continue {
			continue
		} else if action == Stop {
			break
		}
	}

	if err != nil {
		return host, false, buf, err
	}

	al := filters.al
	match := al.MatchDomain(host)

	return host, match, buf, err
}
