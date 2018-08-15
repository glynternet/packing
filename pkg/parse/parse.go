package parse

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

func NewPrefixedParser(prefix string) func(string) (string, error) {
	return func(s string) (string, error) {
		ok := strings.HasPrefix(s, prefix)
		if !ok {
			return "", fmt.Errorf("not prefixed with %q", prefix)
		}
		return s[len(prefix):], nil
	}
}

func NotEmpty(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string given")
	}
	return s, nil
}
