package list

import (
	"bufio"
	"io"
	"strings"

	"github.com/glynternet/packing/pkg/api"
	"github.com/pkg/errors"
)

// ParseContentsDefinition loads the ContentsDefinition of a single list from a Reader
func ParseContentsDefinition(r io.Reader) (api.Contents, error) {
	const referencePrefix = "ref:"
	var refs []string
	var itemNames Items
	p := ProcessorGroup{
		emptyStringCheck,
		CommentProcessor(),
		ReferenceParser(&refs, referencePrefix),
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
		Refs:  refs,
		Items: itemNames,
	}, nil
}

func emptyStringCheck(s string) (bool, error) {
	if s == "" {
		return true, nil
	}
	return false, nil
}
