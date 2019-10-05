package main

import (
	"context"
	"io"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/glynternet/packing/internal/write"
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/client"
	"github.com/glynternet/packing/pkg/cmd"
	"github.com/glynternet/packing/pkg/grpc"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func buildCmdTree(logger *log.Logger, w io.Writer, rootCmd *cobra.Command) {
	viper.SetEnvPrefix("packing")

	const (
		keyServerHost = "server-host"
		keyServerPort = "server-port"
	)

	trip := &cobra.Command{
		Use:  "trip",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			trip := args[0]

			f, err := os.Open(trip)
			if err != nil {
				return errors.Wrapf(err, "opening file at path:%q", trip)
			}
			seed, err := getContentsDefinitionSeed(logger, f)
			if err != nil {
				return errors.Wrap(err, "getting contents definition seed")
			}
			addr := viper.GetString(keyServerHost) + ":" +
				strconv.FormatUint(uint64(viper.GetInt64(keyServerPort)), 10)

			conn, err := grpc.GetGRPCConnection(addr)
			if err != nil {
				return errors.Wrapf(err, "getting new GRPC connection for %q", addr)
			}

			gs, err := client.GetGraph(context.Background(), conn, seed)
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
		},
	}

	trip.Flags().String(keyServerHost, "", "packing server host")
	trip.Flags().Uint(keyServerPort, 3865, "packing server port")
	cmd.MustBindPFlags(logger, trip)
	rootCmd.AddCommand(trip)
}

func getContentsDefinitionSeed(logger *log.Logger, rc io.ReadCloser) (api.ContentsDefinition, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.ContentsDefinition{}, errors.Wrap(err, "parsing contents definition")
	}
	return root, errors.Wrap(rc.Close(), "closing route definition reader")
}
