package main

import (
	"context"
	"io"
	"os"
	"strconv"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/client"
	"github.com/glynternet/packing/pkg/cmd"
	"github.com/glynternet/packing/pkg/graph"
	"github.com/glynternet/packing/pkg/grpc"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/render"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func buildCmdTree(logger log.Logger, w io.Writer, rootCmd *cobra.Command) {
	viper.SetEnvPrefix("packing")

	const (
		keyServerHost = "server-host"
		keyServerPort = "server-port"
	)

	var (
		includeEmptyParentGroups bool
		includeGroupReferences bool
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
			seed, err := getContentsDefinitionSeed(f)
			if err != nil {
				return errors.Wrap(err, "getting contents definition seed")
			}
			addr := viper.GetString(keyServerHost) + ":" +
				strconv.FormatUint(uint64(viper.GetInt64(keyServerPort)), 10)

			conn, err := grpc.GetGRPCConnection(addr)
			if err != nil {
				return errors.Wrapf(err, "getting new GRPC connection for %q", addr)
			}

			gs, err := client.GetGroups(context.Background(), conn, seed)
			if err != nil {
				return errors.Wrap(err, "getting graph")
			}

			return errors.Wrap(render.SortedMarkdownRenderer{
				IncludeEmptyParentGroups:includeEmptyParentGroups,
				IncludeGroupReferences:includeGroupReferences,
				Writer: w,
			}.Render(graph.From(gs)), "rendering graph")
		},
	}

	trip.Flags().String(keyServerHost, "", "packing server host")
	trip.Flags().Uint(keyServerPort, 3865, "packing server port")
	trip.Flags().BoolVar(&includeEmptyParentGroups, "include-empty-parent-groups", false,
		"Provide this flag to render groups that consist only of groups.")
	trip.Flags().BoolVar(&includeGroupReferences, "include-group-references", false,
		"Provide this flag to render references to groups that contain other groups.")
	cmd.MustBindPFlags(logger, trip)
	rootCmd.AddCommand(trip)
}

func getContentsDefinitionSeed(rc io.ReadCloser) (api.ContentsDefinition, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.ContentsDefinition{}, errors.Wrap(err, "parsing contents definition")
	}
	return root, errors.Wrap(rc.Close(), "closing route definition reader")
}
