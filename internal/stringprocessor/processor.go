package stringprocessor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/glynternet/packing/pkg/parse"
)

type Processor func(string) error

type Group []Processor

func (g Group) Process(s string) error {
	for _, processFn := range g {
		err := processFn(s)
		if err == nil {
			return err
		}
		//TODO: use multi error here?
	}
	return fmt.Errorf("unable to Process string:%q", s)
}

func ItemNamesProcessor(names *[]string) Processor {
	return func(s string) error {
		name, err := parse.NotEmpty(s)
		if err == nil {
			*names = append(*names, name)
		}
		return err
	}
}

func GroupNamesProcessor(names *[]string, listNamePrefix string) Processor {
	return func(s string) error {
		groupNameParseFn := parse.NewPrefixedParser(listNamePrefix)
		name, err := groupNameParseFn(s)
		if err == nil {
			*names = append(*names, name)
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
