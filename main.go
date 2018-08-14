package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	gpath "path"
	"strings"

	"github.com/glynternet/packing/internal/stringprocessor"
	"github.com/glynternet/packing/internal/write"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: packing PACKING_FILE LISTS_DIRECTORY")
		return
	}

	out := os.Stdout
	logger := log.New(out, "", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)

	path := os.Args[1]
	listsDir := os.Args[2]
	err := run(path, listsDir, logger, out)
	if err != nil {
		fmt.Fprintf(out, "%v", err)
		os.Exit(1)
	}
}

type listContents struct {
	sublistKeys []string
	items       []string
}

type infoLoader interface {
	load(string) (listContents, error)
}

type fileInfoLoader struct {
	parentDir string
	*log.Logger
}

func (fil fileInfoLoader) load(key string) (listContents, error) {
	path := gpath.Join(string(fil.parentDir), key)
	contents, err := loadContents(path, fil.Logger)
	err = errors.Wrapf(err, "loading contents from path:%q", path)
	return contents, err
}

func recursiveGroupLoad(keys []string, logger *log.Logger, loader infoLoader, groups map[string]list.Group) error {
	if len(keys) == 0 {
		return nil
	}
	var listNames []string
	for _, key := range keys {
		if _, ok := groups[key]; ok {
			// skip if exists
			continue
		}

		contents, err := loader.load(key)
		if err != nil {
			return errors.Wrap(err, "loading info")
		}

		if len(contents.items) > 0 {
			groups[key] = list.Group{
				Name:  key,
				Items: contents.items,
			}
		}
		listNames = append(listNames, contents.sublistKeys...)
	}
	return recursiveGroupLoad(listNames, logger, loader, groups)
}

func run(path string, listsDir string, logger *log.Logger, w io.Writer) error {
	// load data in root file
	cs, err := loadContents(path, logger)
	if err != nil {
		return errors.Wrap(err, "getting root list")
	}

	groups := make(map[string]list.Group)
	if len(cs.items) > 0 {
		name := "Individual Items"
		groups[name] = list.Group{
			Name:  name,
			Items: cs.items,
		}
	}

	loader := fileInfoLoader{
		parentDir: listsDir,
		Logger:    logger,
	}

	err = recursiveGroupLoad(cs.sublistKeys, logger, loader, groups)
	if err != nil {
		return errors.Wrap(err, "loading groups recursively")
	}

	for _, g := range groups {
		write.Group(w, g)
		write.GroupBreak(w)
	}

	return err
}

func loadContents(path string, logger *log.Logger) (listContents, error) {
	lines, err := getFileLines(path, logger)
	if err != nil {
		return listContents{}, errors.Wrapf(err, "getting lines of file at path:%q", path)
	}

	ls, is, err := processLines(lines)
	err = errors.Wrap(err, "processing lines")
	return listContents{
		sublistKeys: ls,
		items:       is,
	}, err
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

func processLines(lines []string) (lists, items []string, err error) {
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
			return nil, nil, errors.Wrapf(err, "processing line:%q", line)
		}
	}

	return listNames, itemNames, err
}

func emptyStringCheck(s string) error {
	if s == "" {
		return nil
	}
	return fmt.Errorf("string:%q is not an empty string", s)
}
