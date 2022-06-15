package parse

import (
	"strings"

	"github.com/pkg/errors"
)

// NewPrefixedParser generates a Parser that expects to parse a string that is prefixed by the given prefix.
// An error is returned by the Parser if the string to parse does not start with the prefix.
func NewPrefixedParser(prefix string) func(string) (string, bool) {
	return func(s string) (string, bool) {
		ok := strings.HasPrefix(s, prefix)
		if !ok {
			return "", ok
		}
		return s[len(prefix):], ok
	}
}

// NotEmpty generates a Parser that returns the given string if it is not empty, returning an error otherwise.
func NotEmpty(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string given")
	}
	return s, nil
}
