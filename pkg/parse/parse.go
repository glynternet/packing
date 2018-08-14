package parse

import (
	"strings"
	"fmt"
)

func NewPrefixParser(prefix string) func(string) (string, error) {
	return func(s string) (string, error) {
		ok := strings.HasPrefix(s, prefix)
		if !ok {
			return "", fmt.Errorf("not prefixed with %q", prefix)
		}
		return s[len(prefix):], nil
	}
}

func ParseItem(s string) (string, error) {
	return s, nil
}
