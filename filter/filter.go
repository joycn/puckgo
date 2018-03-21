package filter

import (
	"bufio"
	"fmt"
	"github.com/joycn/puckgo/datasource"
	"io"
)

type FilterAction int

type Buffer interface {
	Write(io.Writer) error
}

const (
	Continue FilterAction = iota
	Stop
	Again
)

type FilterFunc func(r *bufio.Reader) (string, FilterAction, Buffer, error)

type Filter struct {
	Func func(r *bufio.Reader) (string, FilterAction, Buffer, error)
	Name string
}

type Filters struct {
	m            map[string]*Filter
	MatchActions datasource.MatchActions
}

func NewFilters(ma datasource.MatchActions) *Filters {
	filters := &Filters{MatchActions: ma}
	filters.m = make(map[string]*Filter)
	return filters
}

func (filters *Filters) AddFilter(f *Filter) error {
	n := f.Name
	if _, ok := filters.m[n]; !ok {
		filters.m[n] = f
		return nil
	}
	return fmt.Errorf("filter exist")
}

func (filters *Filters) RemoveFilter(f *Filter) error {
	n := f.Name
	if _, ok := filters.m[n]; ok {
		delete(filters.m, n)
		return nil
	}
	return fmt.Errorf("filter not exist")
}

func (filters *Filters) ExecFilters(r *bufio.Reader) (datasource.MatchAction, Buffer, error) {
	var (
		host   string
		action FilterAction
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
		return datasource.Default, buf, err
	}

	matchAction, err := datasource.Match(host, filters.MatchActions)

	return matchAction, buf, err
}
