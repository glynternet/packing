package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/internal/write"
	"github.com/glynternet/packing/pkg/list"
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
		fmt.Fprintf(os.Stderr, "%v", err)
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

	loader := file.Getter{
		DirPath: listsDir,
		Logger:  logger,
	}

	err = load.GroupsRecursively(cs.SublistKeys, logger, loader, groups)
	if err != nil {
		return errors.Wrap(err, "loading groups recursively")
	}

	if len(cs.Items) > 0 {
		write.Group(w, list.Group{
			Name:  "Individual Items",
			Items: cs.Items,
		})
		write.GroupBreak(w)
	}

	for _, g := range groups {
		write.Group(w, g)
		write.GroupBreak(w)
	}

	return err
}
