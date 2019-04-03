package file

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/glynternet/packing/internal/stringprocessor"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

type GroupGetter struct {
	DirPath string
	*log.Logger
}

func (gg GroupGetter) GetGroup(key string) (list.Group, error) {
	p := path.Join(string(gg.DirPath), key)
	contents, err := LoadGroup(p, gg.Logger)
	err = errors.Wrapf(err, "loading contents from p:%q", p)
	return contents, err
}

// LoadGroup loads a single Group from a file at path
func LoadGroup(path string, logger *log.Logger) (list.Group, error) {
	lines, err := getFileLines(path, logger)
	if err != nil {
		return list.Group{}, errors.Wrapf(err, "getting lines of file at path:%q", path)
	}

	group, err := processLines(lines)
	err = errors.Wrap(err, "processing lines")
	return group, err
}

func getFileLines(path string, logger *log.Logger) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening file at path:%q", path)
	}

	defer func() {
		cErr := file.Close()
		if cErr == nil {
			return
		}
		if err == nil {
			err = cErr
			return
		}
		logger.Println(errors.Wrap(cErr, "closing packing file"))
	}()

	lines, err := getLines(file)
	err = errors.Wrap(err, "getting lines")
	return lines, err
}

func getLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lines = append(lines, line)
	}
	return lines, errors.Wrap(scanner.Err(), "scanning file")
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
