package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"github.com/glynternet/packing/internal/load"
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
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func run(path string, groupsDir string, logger *log.Logger, w io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "opening file at path:%q", path)
	}

	root, err := storage.LoadGroup(f)
	if err != nil {
		return errors.Wrap(err, "getting root list")
	}

	groups := make(map[string]list.Group)

	loader := storage.GroupGetter{
		GetReadCloser: file.ReadCloserGetter(groupsDir),
		Logger:        logger,
	}

	groups, err = load.Groups(root.GroupKeys, logger, loader)
	if err != nil {
		return errors.Wrap(err, "loading groups recursively")
	}

	if len(root.Items) > 0 {
		write.Group(w, list.Group{
			Name:  "Individual Items",
			Items: root.Items,
		})
		write.GroupBreak(w)
	}

	var gs []list.Group
	for _, g := range groups {
		gs = append(gs, g)
	}

	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Name < gs[j].Name
	})

	for _, g := range gs {
		write.Group(w, g)
		write.GroupBreak(w)
	}

	return err
}
