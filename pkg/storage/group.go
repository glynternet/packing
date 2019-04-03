package storage

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/glynternet/packing/internal/stringprocessor"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

type GroupGetter struct {
	GetReadCloser ReadCloserGetter
	*log.Logger
}

type ReadCloserGetter func(key string) (io.ReadCloser, error)

func (gg GroupGetter) GetGroup(key string) (list.Group, error) {
	f, err := gg.GetReadCloser(key)
	if err != nil {
		return list.Group{}, errors.Wrapf(err, "getting ReadCloser for key:%q", key)
	}

	defer func() {
		cErr := f.Close()
		if cErr == nil {
			return
		}
		if err == nil {
			err = cErr
			return
		}
		gg.Logger.Println(errors.Wrap(cErr, "closing group ReadCloser"))
	}()

	contents, err := LoadGroup(f)
	err = errors.Wrapf(err, "loading group for key:%q", key)
	return contents, err
}

// LoadGroup loads a single Group from a Reader
func LoadGroup(r io.Reader) (list.Group, error) {
	lines, err := readAllLines(r)
	if err != nil {
		return list.Group{}, errors.Wrap(err, "reading lines")
	}

	group, err := processLines(lines)
	err = errors.Wrap(err, "processing lines")
	return group, err
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

func processLines(lines []string) (list.Group, error) {
	const groupNamePrefix = "group:"
	var groupNames []string
	var itemNames []string

	p := stringprocessor.Group{
		stringprocessor.GroupNamesProcessor(&groupNames, groupNamePrefix),
		stringprocessor.ItemNamesProcessor(&itemNames),
		emptyStringCheck,
	}

	for _, line := range lines {
		err := p.Process(line)
		if err != nil {
			return list.Group{}, errors.Wrapf(err, "processing line:%q", line)
		}
	}

	return list.Group{
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
