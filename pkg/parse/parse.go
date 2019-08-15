package parse

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// NewPrefixedParser generates a Parser that expects to parse a string that is prefixed by the given prefix.
// An error is returned by the Parser if the string to parse does not start with the prefix.
func NewPrefixedParser(prefix string) func(string) (string, error) {
	return func(s string) (string, error) {
		ok := strings.HasPrefix(s, prefix)
		if !ok {
			return "", fmt.Errorf("not prefixed with %q", prefix)
		}
		return s[len(prefix):], nil
	}
}

// NotEmpty generates a Parser that returns the given string if it is not empty, returning an error otherwise.
func NotEmpty(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string given")
	}
	return s, nil
}
