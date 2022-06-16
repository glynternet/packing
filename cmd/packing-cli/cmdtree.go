package main

import (
	"bytes"
	"encoding/json"
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

const defaultAddr = "http://localhost"

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

	selection := &cobra.Command{
		Use:  "selection",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selection := args[0]

			f, err := os.Open(selection)
			if err != nil {
				return errors.Wrapf(err, "opening file at path:%q", selection)
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

	selection.Flags().String(keyServerHost, "", "packing server host")
	selection.Flags().Uint(keyServerPort, 3865, "packing server port")
	selection.Flags().BoolVar(&includeEmptyParentGroups, "include-empty-parent-groups", false,
		"Provide this flag to render groups that consist only of groups.")
	selection.Flags().BoolVar(&includeGroupReferences, "include-group-references", false,
		"Provide this flag to render references to groups that contain other groups.")
	selection.Flags().StringVar(&renderer, keyRenderer, "html", "renderer to use: markdown or html")
	cmd.MustBindPFlags(logger, selection)
	rootCmd.AddCommand(selection)

	ref := &cobra.Command{
		Use:   "reference <reference>",
		Args:  cobra.ExactArgs(1),
		Short: "Query a reference",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := viper.GetString(keyServerHost) + ":" +
				strconv.FormatUint(uint64(viper.GetInt64(keyServerPort)), 10)

			gs, err := client.GetGroups(logger, addr, api.Contents{
				Refs: []string{args[0]},
			})
			if err != nil {
				return errors.Wrap(err, "getting graph")
			}

			// TODO(glynternet): add renderer option here
			out, err := json.Marshal(gs)
			if err != nil {
				return fmt.Errorf("marshaling response to json: %w", err)
			}

			_, err = w.Write(out)
			return errors.Wrap(err, "writing result to output")
		},
	}
	ref.Flags().String(keyServerHost, defaultAddr, "packing server host")
	ref.Flags().Uint(keyServerPort, 3865, "packing server port")
	cmd.MustBindPFlags(logger, ref)
	rootCmd.AddCommand(ref)
}

type Renderer func(w io.Writer, group []graph.Group) error

func getRenderer(renderer string, includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error) {
	switch renderer {
	case "json":
		return func(w io.Writer, groups []graph.Group) error {
			out, err := json.Marshal(groups)
			if err != nil {
				return fmt.Errorf("marshaling response to json: %w", err)
			}

			_, err = w.Write(out)
			return errors.Wrap(err, "writing result to output")
		}, nil
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
	case "item-list":
		return func(w io.Writer, groups []graph.Group) error {
			for _, group := range groups {
				groupPrefix := group.Group.Name + ":"
				for _, item := range group.Group.Contents.Items {
					if _, err := fmt.Fprintln(w, groupPrefix+item); err != nil {
						return err
					}
				}
			}
			return nil
		}, nil
	}
	return nil, fmt.Errorf(`unsupported renderer:%q, supported renderers are "markdown", "html" and "item-list"`, renderer)
}

func getContentsDefinitionSeed(rc io.ReadCloser) (api.Contents, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.Contents{}, errors.Wrap(err, "parsing contents definition")
	}
	return root, errors.Wrap(rc.Close(), "closing route definition reader")
}
