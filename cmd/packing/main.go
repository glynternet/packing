package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/internal/service"
	"github.com/glynternet/packing/internal/write"
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/config"
	grpc2 "github.com/glynternet/packing/pkg/grpc"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/glynternet/packing/pkg/storage/file"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
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
	gs, err := getFullPackingGraph(logger, conf, seed)
	if err != nil {
		return errors.Wrap(err, "getting full packing graph")
	}

	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Name < gs[j].Name
	})

	for _, g := range gs {
		if len(g.Contents.Items) == 0 {
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

func getFullPackingGraph(logger *log.Logger, conf config.Run, seed api.ContentsDefinition) ([]api.Group, error) {
	s := service.GroupsService{
		Logger: logger,
		Loader: load.Loader{
			ContentsDefinitionGetter: storage.ContentsDefinitionGetter{
				GetReadCloser: file.ReadCloserGetter(conf.GroupsDir),
				Logger:        logger,
			},
		}}

	lis, err := net.Listen("tcp", "")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	sErr := make(chan error)
	cErr := make(chan error)
	gs := make(chan []api.Group)

	go func(errCh chan<- error) {
		logger.Printf("starting server at %s", lis.Addr())
		errCh <- serve(lis, &s)
	}(sErr)

	go func(gsCh chan<- []api.Group, sErr, cErr chan<- error) {
		defer close(gsCh)
		logger.Printf("requesting groups at %s", lis.Addr())
		gs, err := getGraph(context.Background(), lis.Addr().String(), seed)
		cErr <- err
		close(cErr)
		close(sErr)
		gsCh <- gs
	}(gs, sErr, cErr)

	for sErr != nil || cErr != nil {
		select {
		case err, ok := <-sErr:
			if !ok {
				sErr = nil
				continue
			}
			if err != nil {
				return nil, errors.Wrap(err, "server error")
			}
		case err, ok := <-cErr:
			if !ok {
				cErr = nil
				continue
			}
			if err != nil {
				return nil, errors.Wrap(err, "client error")
			}
		}
	}

	return <-gs, nil
}

func getGraph(ctx context.Context, addr string, seed api.ContentsDefinition) ([]api.Group, error) {
	conn, err := grpc2.GetGRPCConnection(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting new GRPC connection")
	}
	groups, err := api.NewGroupsServiceClient(conn).GetGroups(ctx, &seed)
	if err != nil {
		return nil, errors.Wrap(err, "getting groups")
	}
	var gs []api.Group
	for {
		group, err := groups.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "receiving group")
		}
		gs = append(gs, *group)
	}
	return gs, nil
}

func serve(lis net.Listener, server api.GroupsServiceServer) error {
	return errors.Wrap(newServer(server).Serve(lis), "serving groups service")
}

func newServer(gss api.GroupsServiceServer) *grpc.Server {
	srv := grpc.NewServer()
	api.RegisterGroupsServiceServer(srv, gss)
	return srv
}

func getContentsDefinitionSeed(logger *log.Logger, rc io.ReadCloser) (api.ContentsDefinition, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.ContentsDefinition{}, errors.Wrap(err, "parsing contents definition")
	}
	return root, errors.Wrap(rc.Close(), "closing route definition reader")
}
