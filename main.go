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

func run(path string, logger *log.Logger, out io.Writer) error {
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

	err = processLines(lines)

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

func processLines(lines []string) error {
	const listIdentifier = "list:"
	listNameParseFn := parse.NewPrefixParser(listIdentifier)

	for _, line := range lines {
		err := processLine(line, listNameParseFn, parse.ParseItem)
		if err != nil {
			return errors.Wrapf(err, "processing line:%q", line)
		}
	}
	return nil
}

type parseFn func(string) (string, error)

func processLine(line string, listNameParseFn parseFn, itemNameParser parseFn) error {
	if line == "" {
		return nil
	}

	ln, err := listNameParseFn(line)
	if err == nil {
		fmt.Println("it's a list! " + ln)
		return nil
	}

	in, err := itemNameParser(line)
	if err == nil {
		fmt.Println("it's an item! " + in)
		return nil
	}

	return err
}