package stringprocessor

import (
	"fmt"

	"github.com/glynternet/packing/pkg/parse"
)

type Processor func(string) error

type Group []Processor

func (p Group) Process(s string) error {
	for _, fn := range p {
		err := fn(s)
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

func ListNamesProcessor(names *[]string, listNamePrefix string) Processor {
	return func(s string) error {
		listNameParseFn := parse.NewPrefixedParser(listNamePrefix)
		name, err := listNameParseFn(s)
		if err == nil {
			*names = append(*names, name)
		}
		return err
	}
}
