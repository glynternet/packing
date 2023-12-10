package list

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/glynternet/packing/pkg/api"
	"github.com/pkg/errors"
)

const groupNamePrefix = "ref:"

// ParseContentsDefinition loads the ContentsDefinition of a single list from a Reader
func ParseContentsDefinition(r io.Reader) (api.Contents, error) {
	var groupNames GroupKeys
	var itemNames Items
	p := ProcessorGroup{
		emptyStringCheck,
		CommentProcessor(),
		GroupKeysProcessor(&groupNames, groupNamePrefix),
		ItemNamesProcessor(&itemNames),
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if err := p.Process(line); err != nil {
			return api.Contents{}, errors.Wrapf(err, "processing line:%q", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return api.Contents{}, errors.Wrap(err, "scanning lines")
	}

	return api.Contents{
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
