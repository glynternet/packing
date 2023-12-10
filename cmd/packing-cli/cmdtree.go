package main

import (
	"io"
	"os"
	"strconv"

	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/client"
	"github.com/glynternet/packing/pkg/cmd"
	"github.com/glynternet/packing/pkg/graph"
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
			gs, err := client.GetGroups(logger, addr, seed)
			if err != nil {
				return errors.Wrap(err, "getting graph")
			}

			return errors.Wrap(render.SortedMarkdownRenderer{Writer: w}.Render(graph.From(gs)), "rendering graph")
		},
	}

	trip.Flags().String(keyServerHost, "http://localhost", "packing server host")
	trip.Flags().Uint(keyServerPort, 3865, "packing server port")
	cmd.MustBindPFlags(logger, trip)
	rootCmd.AddCommand(trip)
}

func getContentsDefinitionSeed(rc io.ReadCloser) (api.Contents, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.Contents{}, errors.Wrap(err, "parsing contents definition")
	}
	return root, errors.Wrap(rc.Close(), "closing route definition reader")
}
