package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/glynternet/packing/internal/write"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/glynternet/packing/pkg/storage/file"
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

func run(path string, listsDir string, logger *log.Logger, w io.Writer) error {
	// Get data in root file
	cs, err := file.LoadContents(path, logger)
	if err != nil {
		return errors.Wrap(err, "getting root list")
	}

	groups := make(map[string]list.Group)
	if len(cs.Items) > 0 {
		name := "Individual Items"
		groups[name] = list.Group{
			Name:  name,
			Items: cs.Items,
		}
	}

	loader := file.Getter{
		DirPath: listsDir,
		Logger:  logger,
	}

	err = recursiveGroupLoad(cs.SublistKeys, logger, loader, groups)
	if err != nil {
		return errors.Wrap(err, "loading groups recursively")
	}

	for _, g := range groups {
		write.Group(w, g)
		write.GroupBreak(w)
	}

	return err
}

func recursiveGroupLoad(keys []string, logger *log.Logger, contentsGetter storage.ListContentsGetter, groups map[string]list.Group) error {
	if len(keys) == 0 {
		return nil
	}
	var listNames []string
	for _, key := range keys {
		if _, ok := groups[key]; ok {
			// skip if exists
			continue
		}

		contents, err := contentsGetter.Get(key)
		if err != nil {
			return errors.Wrap(err, "loading info")
		}

		if len(contents.Items) > 0 {
			groups[key] = list.Group{
				Name:  key,
				Items: contents.Items,
			}
		}
		listNames = append(listNames, contents.SublistKeys...)
	}
	return recursiveGroupLoad(listNames, logger, contentsGetter, groups)
}
