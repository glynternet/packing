package list

import (
	"bufio"
	"io"
	"strings"

	"github.com/glynternet/packing/pkg/api"
	"github.com/pkg/errors"
)

const referencePrefix = "ref"
const requirementPrefix = "req"

// ParseContentsDefinition loads the ContentsDefinition of a single list from a Reader
func ParseContentsDefinition(r io.Reader) (api.Contents, error) {
	var refs References
	var reqs References
	var itemNames Items
	p := ProcessorGroup{
		emptyStringCheck,
		TaggedLineParser(&refs, &reqs),
		ItemNamesProcessor(&itemNames),
	}

	scanner := bufio.NewScanner(r)
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		if commentStart := strings.IndexRune(line, '#'); commentStart != -1 {
			line = line[:commentStart]
		}
		line = strings.TrimSpace(line)
		if err := p.Process(line); err != nil {
			return api.Contents{}, errors.Wrapf(err, "processing line %d: %q", lineNum, line)
		}
	}

	return api.Contents{
		Refs:     refs,
		Items:    itemNames,
		Requires: reqs,
	}, errors.Wrap(scanner.Err(), "scanning lines")
}

func emptyStringCheck(s string) (bool, error) {
	if s == "" {
		return true, nil
	}
	return false, nil
}
