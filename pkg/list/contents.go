package list

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

type ContentsDefinition struct {
	GroupKeys GroupKeys
	Items     Items
}

// ParseContentsDefinition loads the ContentsDefinition of a single list from a Reader
func ParseContentsDefinition(r io.Reader) (ContentsDefinition, error) {
	lines, err := readAllLines(r)
	if err != nil {
		return ContentsDefinition{}, errors.Wrap(err, "reading lines")
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

func processLines(lines []string) (ContentsDefinition, error) {
	const groupNamePrefix = "group:"
	var groupNames GroupKeys
	var itemNames Items

	p := ProcessorGroup{
		emptyStringCheck,
		CommentProcessor(),
		GroupNamesProcessor(&groupNames, groupNamePrefix),
		ItemNamesProcessor(&itemNames),
	}

	for _, line := range lines {
		err := p.Process(line)
		if err != nil {
			return ContentsDefinition{}, errors.Wrapf(err, "processing line:%q", line)
		}
	}

	return ContentsDefinition{
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
