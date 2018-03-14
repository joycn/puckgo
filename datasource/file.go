package datasource

import (
	"bufio"
	"os"
	"strings"
)

// MatchActionsFromFile get urlactions from path
func MatchActionsFromFile(path string) (MatchActions, error) {
	ret := make(MatchActions)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		url, err := reader.ReadString('\n')
		url = strings.TrimSpace(url)
		if err != nil {
			break
		}
		ret[url] = Except
	}
	return ret, nil
}
