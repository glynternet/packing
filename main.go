package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"github.com/pkg/errors"
	"io"
	"strings"
	"github.com/glynternet/packing/pkg/parse"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: packing PACKING_FILE")
		return
	}

	out := os.Stdout
	logger := log.New(out, "", log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)

	path := os.Args[1]
	err := run(path, logger, out)
	if err != nil {
		fmt.Fprintf(out, "%v", err)
		os.Exit(1)
	}
}

func run(path string, logger *log.Logger, w io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "opening filer at path:%s", path)
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

	if len(is) > 0 {
		writeTitle(w, "Individual Items")
		for _, item := range is {
			fmt.Fprintln(w, item)
		}
		writeGroupBreak(w)
	}

	for _, listName := range ls {
		writeTitle(w, listName)
		writeGroupBreak(w)
	}

	return err
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

	p := stringProcessorGroup{
		listNamesProcessor(&listNames, listNamePrefix),
		itemNamesProcessor(&itemNames),
		emptyStringCheck,
	}

	for _, line := range lines {
		err := p.process(line)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "processing line:%q", line)
		}
	}

	return listNames, itemNames, err
}

type stringProcessor func(string) error

type stringProcessorGroup []stringProcessor

func (p stringProcessorGroup) process(s string) error {
	for _, fn := range p {
		err := fn(s)
		if err == nil {
			return err
		}
		//TODO: use multi error here?
	}
	return fmt.Errorf("unable to process string:%q", s)
}

func itemNamesProcessor(names *[]string) stringProcessor {
	return func(s string) error {
		name, err := parse.Item(s)
		if err == nil {
			*names = append(*names, name)
		}
		return err
	}
}

func listNamesProcessor(names *[]string, listNamePrefix string) stringProcessor {
	return func(s string) error {
		listNameParseFn := parse.NewPrefixedParser(listNamePrefix)
		name, err := listNameParseFn(s)
		if err == nil {
			*names = append(*names, name)
		}
		return err
	}
}

func emptyStringCheck(s string) error {
	if s == "" {
		return nil
	}
	return fmt.Errorf("string:%q is not an empty string", s)
}
