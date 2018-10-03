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
	if len(cs.Items) > 0 {
		// BUG(glyn): By adding this to the map here, it means that if there is a
		// sublist later in the application called "Individual Items", it will not
		// be added to the map. If we move adding the "Individual Items" to later
		// in the flow, we can check if it exists already, then name it something
		// else that has not yet been taken.
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

	err = load.GroupsRecursively(cs.SublistKeys, logger, loader, groups)
	if err != nil {
		return errors.Wrap(err, "loading groups recursively")
	}

	for _, g := range groups {
		write.Group(w, g)
		write.GroupBreak(w)
	}

	return err
}
