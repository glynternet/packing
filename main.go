package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"github.com/pkg/errors"
	"io"
	"strings"
	"path"
	"github.com/glynternet/packing/internal/stringprocessor"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/internal/write"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: packing PACKING_FILE LISTS_DIRECTORY")
		return
	}

	out := os.Stdout
	logger := log.New(out, "", log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)

	path := os.Args[1]
	listsDir := os.Args[2]
	err := run(path, listsDir, logger, out)
	if err != nil {
		fmt.Fprintf(out, "%v", err)
		os.Exit(1)
	}
}

type groupLoader func(string) (list.Group, error)

func fileGroupLoader(parentDir string, logger *log.Logger) groupLoader {
	return func(s string) (list.Group, error) {
		path := path.Join(parentDir, s)
		file, err := os.Open(path)
		if err != nil {
			return list.Group{}, errors.Wrapf(err, "opening file at path:%q", path)
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
		if err != nil {
			err = errors.Wrap(err, "getting lines")
			return list.Group{}, err
		}

		return list.Group{
			Name:s,
			Items:lines,
		}, nil
	}
}

func run(path string, listsDir string, logger *log.Logger, w io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "opening file at path:%q", path)
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
	if err != nil {
		err = errors.Wrap(err, "getting lines")
		return err
	}

	ls, is, err := processLines(lines)
	if err != nil {
		err = errors.Wrap(err, "processing lines")
		return err
	}

	var gs []list.Group

	if len(is) > 0 {
		gs = append(gs, list.Group{
			Name:"Individual Items",
			Items:is,
		})
	}

	loadGroup := fileGroupLoader(listsDir, logger)
	errs := make(map[string]error)

	for _, name := range ls {
		g, err := loadGroup(name)
		if err != nil {
			errs[name] = errors.Wrapf(err, "loading list with name:%q", name)
			continue
		}
		gs = append(gs, g)
	}

	if len(errs) > 0 {
		write.Title(w, "Errors")
		for _, err := range errs {
			fmt.Fprintln(w, err)
		}
		write.GroupBreak(w)
	}

	for i, g := range gs {
		write.Group(w, g)
		if i < len(gs) - 1 {
			write.GroupBreak(w)
		}
	}

	return err
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

func processLines(lines []string) (lists, items []string, err error){
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
