package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/internal/write"
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/config"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/glynternet/packing/pkg/storage/file"
	"github.com/pkg/errors"
)

// to be changed using ldflags with the go build command
var version = "unknown"

func main() {
	printVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *printVersion {
		fmt.Println(version)
		return
	}

	if len(os.Args) != 2 && len(os.Args) != 3 {
		fmt.Println("Usage: packing PACKING_FILE GROUPS_DIRECTORY")
		return
	}

	out := os.Stdout
	logger := log.New(out, "", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)

	err := run(config.Run{
		TripPath:  os.Args[1],
		GroupsDir: groupsDir(),
	}, logger, out)
	if err != nil {
		_, pErr := fmt.Fprintf(out, "%v\n", err)
		if pErr != nil {
			_, _ = fmt.Fprintf(out, "%v\n", errors.Wrap(pErr, "writing error to writer"))
		}
		os.Exit(1)
	}
}

func groupsDir() string {
	if len(os.Args) > 2 {
		return os.Args[2]
	}
	return os.Getenv("PACKING_GROUPS_DIR")
}

func run(conf config.Run, logger *log.Logger, w io.Writer) error {
	f, err := os.Open(conf.TripPath)
	if err != nil {
		return errors.Wrapf(err, "opening file at path:%q", conf.TripPath)
	}
	seed, err := getContentsDefinitionSeed(logger, f)
	if err != nil {
		return errors.Wrap(err, "getting contents definition seed")
	}
	loader := storage.ContentsDefinitionGetter{
		GetReadCloser: file.ReadCloserGetter(conf.GroupsDir),
		Logger:        logger,
	}

	groups, err := load.AllGroups(logger, seed, loader)
	if err != nil {
		return errors.Wrap(err, "loading all groups")
	}

	var gs []list.Group
	for _, g := range groups {
		gs = append(gs, g)
	}

	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Name < gs[j].Name
	})

	for _, g := range gs {
		if len(g.Items) == 0 {
			continue
		}
		if err := write.Group(w, g); err != nil {
			return errors.Wrapf(err, "writing Group %q to writer", g)
		}
		if err := write.GroupBreak(w); err != nil {
			return errors.Wrapf(err, "writing GroupBreak %q to writer", g)
		}
	}

	return err
}

func getContentsDefinitionSeed(logger *log.Logger, rc io.ReadCloser) (api.ContentsDefinition, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.ContentsDefinition{}, errors.Wrap(err, "parsing contents definition")
	}
	defer func() {
		cErr := rc.Close()
		if cErr == nil {
			return
		}
		if err != nil {
			logger.Println(errors.Wrap(cErr, "closing route definitin reader"))
			return
		}
		err = cErr
	}()
	return root, err
}
