package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/client"
	"github.com/glynternet/packing/pkg/graph"
	"github.com/glynternet/packing/pkg/inline"
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
		inlineSingleItemGroups   bool
		renderer                 string
	)

	supportedRenderers, getRenderer := rendererFactory()

	selection := &cobra.Command{
		Use:   "selection <file>",
		Args:  cobra.ExactArgs(1),
		Short: "Render a packing list from a local selection file",
		Long: `Render a full packing list from a local selection file.

<file> is a path to a local file describing what you are packing for.
Each line is one of:

  <item>       an item to pack            (e.g. "toothbrush")
  ref:<name>   include a server group by its name
  req:<name>   mark a server group as required
  # ...        a comment (also allowed at the end of a line)

Blank lines are ignored. The file is sent to the packing server, which
recursively expands every ref: against its groups directory. The resulting
list is rendered using --renderer.`,
		// Bind this command's flags to viper here rather than at tree-build time.
		// viper is a global singleton, so binding every command's flags eagerly
		// makes the last-bound command's flags win for shared keys (server-host,
		// server-port), silently ignoring this command's --server-* flags. PreRunE
		// runs only for the command actually being executed.
		PreRunE: func(c *cobra.Command, _ []string) error {
			return viper.BindPFlags(c.Flags())
		},
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

			if inlineSingleItemGroups {
				gs = inline.SingleItemGroups(gs)
			}

			render, err := getRenderer(renderer, includeEmptyParentGroups, includeGroupReferences)
			if err != nil {
				return errors.Wrap(err, "getting renderer")
			}

			return errors.Wrap(render(w, graph.From(gs)), "rendering graph")
		},
	}

	selection.Flags().String(keyServerHost, defaultAddr, "packing server host, e.g. http://localhost")
	selection.Flags().Uint(keyServerPort, 3865, "packing server port")
	selection.Flags().BoolVar(&includeEmptyParentGroups, "include-empty-parent-groups", false,
		"Provide this flag to render groups that consist only of groups.")
	selection.Flags().BoolVar(&includeGroupReferences, "include-group-references", false,
		"Provide this flag to render references to groups that contain other groups.")
	selection.Flags().BoolVar(&inlineSingleItemGroups, "inline-single-item-groups", true,
		"Inline references that resolve to a single item into their parent groups instead of showing a standalone group. Use --inline-single-item-groups=false to keep them as groups.")
	selection.Flags().StringVar(&renderer, keyRenderer, "html", "renderer to use: "+strings.Join(supportedRenderers, ", "))
	rootCmd.AddCommand(selection)

	ref := &cobra.Command{
		Use:   "reference <reference> [<reference>...]",
		Args:  cobra.MinimumNArgs(1),
		Short: "Fetch groups from the server by name",
		Long: `Fetch one or more groups directly from the packing server by name.

Each <reference> is the name (key) of a group hosted by the server — i.e.
the filename of a group in the server's groups directory. Unlike "selection",
this does not read a local file; it looks up the given names, expands them,
and prints the result as JSON.`,
		// See the note on the selection command: bind per-command in PreRunE so
		// this command's --server-* flags are not clobbered by another command's.
		PreRunE: func(c *cobra.Command, _ []string) error {
			return viper.BindPFlags(c.Flags())
		},
		RunE: func(cmd *cobra.Command, keys []string) error {
			addr := viper.GetString(keyServerHost) + ":" +
				strconv.FormatUint(uint64(viper.GetInt64(keyServerPort)), 10)

			gs, err := client.GetGroups(logger, addr, api.Contents{
				Refs: keys,
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
	ref.Flags().String(keyServerHost, defaultAddr, "packing server host, e.g. http://localhost")
	ref.Flags().Uint(keyServerPort, 3865, "packing server port")
	rootCmd.AddCommand(ref)
}

type Renderer func(w io.Writer, group []graph.Group) error

func rendererFactory() ([]string, func(renderer string, includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error)) {
	renderers := map[string]func(includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error){
		"json": func(includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error) {
			return func(w io.Writer, groups []graph.Group) error {
				out, err := json.Marshal(groups)
				if err != nil {
					return fmt.Errorf("marshaling response to json: %w", err)
				}

				_, err = w.Write(out)
				return errors.Wrap(err, "writing result to output")
			}, nil
		},
		"markdown": func(includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error) {
			return render.SortedMarkdownRenderer{
				IncludeEmptyParentGroups: includeEmptyParentGroups,
				IncludeGroupReferences:   includeGroupReferences,
			}.Render, nil
		},
		"html": func(includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error) {
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
		},
		"item-list": func(includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error) {
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
		},
	}
	var supported []string
	for renderer := range renderers {
		supported = append(supported, renderer)
	}
	sort.Strings(supported)
	return supported, func(renderer string, includeEmptyParentGroups, includeGroupReferences bool) (Renderer, error) {
		if r, ok := renderers[renderer]; ok {
			return r(includeEmptyParentGroups, includeGroupReferences)
		}
		return nil, fmt.Errorf(`unsupported renderer:%q, supported renderers are: %s`, renderer, strings.Join(supported, ", "))
	}
}

func getContentsDefinitionSeed(rc io.ReadCloser) (api.Contents, error) {
	root, err := list.ParseContentsDefinition(rc)
	if err != nil {
		return api.Contents{}, errors.Wrap(err, "parsing contents definition")
	}
	return root, errors.Wrap(rc.Close(), "closing route definition reader")
}
