package datasource

import (
	"fmt"
	"strings"
)

// MatchAction action for matched url
type MatchAction bool

const (
	// Except forward to exceptive server
	Except MatchAction = true
	// Default forward to default server
	Default MatchAction = false
)

// MatchActions map info for dns name and actions
type MatchActions map[string]MatchAction

// GetMatchActions get matchactions from source
func GetMatchActions(source string) (MatchActions, error) {
	tokens := strings.SplitN(source, ":", 2)
	if len(tokens) != 2 {
		return nil, fmt.Errorf("datasource format error")
	}

	switch tokens[0] {
	case "file":
		return MatchActionsFromFile(tokens[1])
	case "etcd":
	}
	return make(MatchActions), nil
}

func Match(name string, ma MatchActions) (MatchAction, error) {
	name = strings.TrimSuffix(name, ".")
	var tokens []string
	for {
		if action, ok := ma[name]; ok {
			return action, nil
		}
		tokens = strings.SplitN(name, ".", 2)
		if len(tokens) == 1 {
			break
		}
		name = tokens[1]
	}
	return Default, fmt.Errorf("not match")
}
