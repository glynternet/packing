package parse

import (
	"strings"
	"fmt"
	"github.com/pkg/errors"
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

func Item(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string given")
	}
	return s, nil
}
