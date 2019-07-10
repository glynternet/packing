package stringprocessor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/parse"
)

// Processor should return an error if the string cannot be processed as the given type
type Processor func(string) error

// Group is a group of processors
type Group []Processor

// Process applies each Processor in the Group in order until either:
// - no error is given, in which case nil is returned.
// - all Processors have been applied, in which case the last is returned
func (g Group) Process(s string) error {
	for _, processFn := range g {
		err := processFn(s)
		if err == nil {
			return err
		}
		//TODO(glynternet): use multi error here?
	}
	return fmt.Errorf("unable to Process string:%q", s)
}

func ItemNamesProcessor(items *list.Items) Processor {
	// TODO(glynternet): does this need to be improved to ignore groups and other cases?
	return func(s string) error {
		name, err := parse.NotEmpty(s)
		if err == nil {
			*items = append(*items, list.Item(name))
		}
		return err
	}
}

func GroupNamesProcessor(names *list.GroupKeys, listNamePrefix string) Processor {
	return func(s string) error {
		groupNameParseFn := parse.NewPrefixedParser(listNamePrefix)
		name, err := groupNameParseFn(s)
		if err == nil {
			*names = append(*names, list.GroupKey(name))
		}
		return err
	}
}

func CommentProcessor() Processor {
	return func(s string) error {
		if strings.HasPrefix(strings.TrimSpace(s), "#") {
			return nil
		}
		return errors.New("given string is not a comment")
	}
}
