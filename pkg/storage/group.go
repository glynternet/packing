package storage

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/glynternet/packing/internal/stringprocessor"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

// LoadContents loads the contents of a single list from a Reader
func LoadContents(r io.Reader) (list.Contents, error) {
	lines, err := readAllLines(r)
	if err != nil {
		return list.Contents{}, errors.Wrap(err, "reading lines")
	}

	cs, err := processLines(lines)
	err = errors.Wrap(err, "processing lines")
	return cs, err
}

func readAllLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lines = append(lines, line)
	}
	return lines, errors.Wrap(scanner.Err(), "scanning lines")
}

func processLines(lines []string) (list.Contents, error) {
	const groupNamePrefix = "group:"
	var groupNames list.GroupKeys
	var itemNames list.Items

	p := stringprocessor.Group{
		emptyStringCheck,
		stringprocessor.CommentProcessor(),
		stringprocessor.GroupNamesProcessor(&groupNames, groupNamePrefix),
		stringprocessor.ItemNamesProcessor(&itemNames),
	}

	for _, line := range lines {
		err := p.Process(line)
		if err != nil {
			return list.Contents{}, errors.Wrapf(err, "processing line:%q", line)
		}
	}

	return list.Contents{
		GroupKeys: groupNames,
		Items:     itemNames,
	}, nil
}

func emptyStringCheck(s string) error {
	if s == "" {
		return nil
	}
	return fmt.Errorf("string:%q is not an empty string", s)
}
