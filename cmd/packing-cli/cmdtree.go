package main

import (
	"bytes"
	"fmt"
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
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func buildCmdTree(logger log.Logger, w io.Writer, rootCmd *cobra.Command) {
	viper.SetEnvPrefix("packing")

	const (
		keyServerHost = "server-host"
		keyServerPort = "server-port"
		keyRenderer   = "renderer"
	)

	var (
		includeEmptyParentGroups bool
		includeGroupReferences   bool
		renderer                 string
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

			render, err := getRenderer(renderer, includeEmptyParentGroups, includeGroupReferences)
			if err != nil {
				return errors.Wrap(err, "getting renderer")
			}

			return errors.Wrap(render(w, graph.From(gs)), "rendering graph")
		},
	}

	trip.Flags().String(keyServerHost, "http://localhost", "packing server host")
	trip.Flags().Uint(keyServerPort, 3865, "packing server port")
	trip.Flags().BoolVar(&includeEmptyParentGroups, "include-empty-parent-groups", false,
		"Provide this flag to render groups that consist only of groups.")
	trip.Flags().BoolVar(&includeGroupReferences, "include-group-references", false,
		"Provide this flag to render references to groups that contain other groups.")
	trip.Flags().StringVar(&renderer, keyRenderer, "html", "renderer to use: markdown or html")
	cmd.MustBindPFlags(logger, trip)
	rootCmd.AddCommand(trip)
}

type Renderer func(w io.Writer, group []graph.Group) error

func getRenderer(renderer string, includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error) {
	switch renderer {
	case "markdown":
		return render.SortedMarkdownRenderer{
			IncludeEmptyParentGroups: includeEmptyParentGroups,
			IncludeGroupReferences:   includeGroupReferences,
		}.Render, nil
	case "html":
		return func(w io.Writer, group []graph.Group) error {
			var md bytes.Buffer
			mdRenderer := render.SortedMarkdownRenderer{
				IncludeEmptyParentGroups: includeEmptyParentGroups,
				IncludeGroupReferences:   includeGroupReferences,
			}
			if err := mdRenderer.Render(&md, group); err != nil {
				return errors.Wrap(err, "rendering intermediate markdown")
			}

			extensions := parser.CommonExtensions | parser.AutoHeadingIDs
			_, err := w.Write(markdown.ToHTML(md.Bytes(), parser.NewWithExtensions(extensions), nil))
			return errors.Wrap(err, "writing html to writer")
		}, nil
	}
	return nil, fmt.Errorf(`unsupported renderer:%q, supported renderers are "markdown" and "html"`, renderer)
}

func getContentsDefinitionSeed(rc io.ReadCloser) (api.Contents, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.Contents{}, errors.Wrap(err, "parsing contents definition")
	}
	return root, errors.Wrap(rc.Close(), "closing route definition reader")
}
