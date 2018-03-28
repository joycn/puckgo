package filter

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

const (
	// LongestMehodLen max read len to fetch http method
	LongestMehodLen = 9
)

var (
	// HTTPMethod supprted http method
	HTTPMethod = map[string]bool{
		"GET":       true,
		"PUT":       true,
		"POST":      true,
		"COPY":      true,
		"MOVE":      true,
		"LOCK":      true,
		"HEAD":      true,
		"MKCOL":     true,
		"PATCH":     true,
		"TRACE":     true,
		"DELETE":    true,
		"UNLOCK":    true,
		"OPTIONS":   true,
		"PROPFIND":  true,
		"PROPPATCH": true,
	}
)

// NewHTTPFilter create a new http filter filter with http host header
func NewHTTPFilter() *Filter {
	return &Filter{Name: "http", Func: filterByHTTPHost}
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

func filterByHTTPHost(r *bufio.Reader) (string, Action, Buffer, error) {

	firstChar, err := r.Peek(1)

	if err != nil {
		return "", Again, nil, err
	}

	ch := firstChar[0]

	if (ch < 'A' || ch > 'Z') && ch != '_' && ch != '-' {
		return "", Continue, nil, fmt.Errorf("not http")
	}

	prefix, err := r.Peek(LongestMehodLen + 1)
	if err != nil {
		return "", Again, nil, err
	}

	methods := strings.SplitN(string(prefix), " ", 2)

	if len(methods) < 2 {
		return "", Continue, nil, fmt.Errorf("not http")
	} else if _, ok := HTTPMethod[methods[0]]; !ok {
		return "", Continue, nil, fmt.Errorf("not http")
	}

	var request *http.Request
	if request, err = http.ReadRequest(r); err != nil {
		return "", Stop, request, err
	}
	return strings.SplitN(request.Host, ":", 2)[0], Stop, request, nil
}
