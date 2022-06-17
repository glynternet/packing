package list

import (
	"errors"
	"fmt"
	"strings"

	"github.com/glynternet/packing/pkg/parse"
)

// Processor should return true if the item has been processed and should not be processed by any others, or an error if
// the string cannot be processed.
type Processor func(string) (bool, error)

// ProcessorGroup is a group of processors
type ProcessorGroup []Processor

// Process applies each Processor in the Group in order until either:
// - no error is given, in which case nil is returned.
// - all Processors have been applied, in which case the last is returned
func (g ProcessorGroup) Process(s string) error {
	if len(g) == 0 {
		return errors.New("no processors configured")
	}
	for _, processFn := range g {
		done, err := processFn(s)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return fmt.Errorf("no processor matched value: %q", s)
}

// ItemNamesProcessor generates a Processor that parses a line into an api.Item
func ItemNamesProcessor(items *Items) Processor {
	// TODO(glynternet): does this need to be improved to ignore groups and other cases?
	return func(s string) (bool, error) {
		name, err := parse.NotEmpty(s)
		if err == nil {
			*items = append(*items, name)
		}
		return true, err
	}
}

// TaggedLineParser generates a Processor that attempts to parse lines into references and requirements
func TaggedLineParser(refs *References, reqs *References) Processor {
	return func(s string) (bool, error) {
		i := strings.IndexRune(s, ':')
		if i == -1 {
			return false, nil
		}

		tagValue := strings.TrimSpace(s[i+1:])
		switch tag := s[:i]; tag {
		case referencePrefix:
			if tagValue == "" {
				return false, errors.New("empty reference value")
			}
			*refs = append(*refs, tagValue)
		case requirementPrefix:
			if tagValue == "" {
				return false, errors.New("empty requirement value")
			}
			*reqs = append(*reqs, tagValue)
		default:
			return false, fmt.Errorf("unsupported tag prefix: %q", tag)
		}
		return true, nil
	}
}

// CommentProcessor generates a Processor that parses comment lines, returning an error if the given line is not a
// comment
func CommentProcessor() Processor {
	return func(s string) (bool, error) {
		if strings.HasPrefix(strings.TrimSpace(s), "#") {
			return true, nil
		}
		return false, nil
	}
}
