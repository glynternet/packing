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

type groupLoader func(string) (group, error)

func fileGroupLoader(parentDir string, logger *log.Logger) groupLoader {
	return func(s string) (group, error) {
		path := path.Join(parentDir, s)
		file, err := os.Open(path)
		if err != nil {
			return group{}, errors.Wrapf(err, "opening file at path:%q", path)
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
			return group{}, err
		}

		return group{
			name:s,
			items:lines,
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

	var gs []group

	if len(is) > 0 {
		gs = append(gs, group{
			name:"Individual Items",
			items:is,
		})
	}

	loadGroup := fileGroupLoader(listsDir, logger)
	errs := make(map[string]error)

	for _, name := range ls {
		g, err := loadGroup(name)
		if err != nil {
			errs[name] = errors.Wrapf(err, "loading group with name:%q", name)
			continue
		}
		gs = append(gs, g)
	}

	if len(errs) > 0 {
		writeTitle(w, "Errors")
		for _, err := range errs {
			fmt.Fprintln(w, err)
		}
		writeGroupBreak(w)
	}

	for i, g := range gs {
		writeGroup(w, g)
		if i < len(gs) - 1 {
			writeGroupBreak(w)
		}
	}

	return err
}

type group struct {
	name string
	items []string
}

func writeGroup(w io.Writer, g group) {
	name := strings.TrimSpace(g.name)
	if name == "" {
		name = "Unnamed"
	}
	writeTitle(w, name)
	for _, item := range g.items {
		fmt.Fprintln(w, item)
	}
}

func writeTitle(w io.Writer, text string) {
	fmt.Fprintln(w, strings.ToUpper(text))
}

func writeGroupBreak(w io.Writer) {
	const groupBreak = "\n\n"
	fmt.Fprint(w, groupBreak)
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
