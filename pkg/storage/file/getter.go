package file

import (
	"bufio"
	"io"
	"log"
	"os"
	gpath "path"
	"strings"

	"fmt"

	"github.com/glynternet/packing/internal/stringprocessor"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

type Getter struct {
	DirPath string
	*log.Logger
}

func (fil Getter) Get(key string) (list.Contents, error) {
	path := gpath.Join(string(fil.DirPath), key)
	contents, err := LoadContents(path, fil.Logger)
	err = errors.Wrapf(err, "loading contents from path:%q", path)
	return contents, err
}

func LoadContents(path string, logger *log.Logger) (list.Contents, error) {
	lines, err := getFileLines(path, logger)
	if err != nil {
		return list.Contents{}, errors.Wrapf(err, "getting lines of file at path:%q", path)
	}

	cs, err := processLines(lines)
	err = errors.Wrap(err, "processing lines")
	return cs, err
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

func processLines(lines []string) (list.Contents, error) {
	const listNamePrefix = "list:"
	var listNames []string
	var itemNames []string

	p := stringprocessor.Group{
		stringprocessor.ListNamesProcessor(&listNames, listNamePrefix),
		stringprocessor.ItemNamesProcessor(&itemNames),
		emptyStringCheck,
	}

	for _, line := range lines {
		err := p.Process(line)
		if err != nil {
			return list.Contents{}, errors.Wrapf(err, "processing line:%q", line)
		}
	}

	return list.Contents{
		SublistKeys: listNames,
		Items:       itemNames,
	}, nil
}

func emptyStringCheck(s string) error {
	if s == "" {
		return nil
	}
	return fmt.Errorf("string:%q is not an empty string", s)
}
