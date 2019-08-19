package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/glynternet/packing/pkg/api/build"
	"github.com/pkg/errors"
)

// SortedMarkdownRenderer renders a graph to its writer sorted by group name
type SortedMarkdownRenderer struct {
	io.Writer
}

// Render renders a graph to the SortedMarkdownRenderer's writer sorted by group name
func (r SortedMarkdownRenderer) Render(gs []api.Group) error {
	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Name < gs[j].Name
	})

	for _, g := range gs {
		if len(g.Contents.Items) == 0 {
			continue
		}
		if err := r.group(g); err != nil {
			return errors.Wrapf(err, "writing Group %q to writer", g)
		}
		if err := r.groupBreak(); err != nil {
			return errors.Wrapf(err, "writing GroupBreak %q to writer", g)
		}
	}
	return nil
}

func (r SortedMarkdownRenderer) group(g api.Group) error {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		name = "Unnamed"
	}
	if err := r.title(name); err != nil {
		return errors.Wrapf(err, "writing Title %q to writer", name)
	}
	for _, item := range g.Contents.Items {
		if err := r.item(*item); err != nil {
			return errors.Wrapf(err, "writing Item %q to writer", *item)
		}
	}
	return nil
}

func (r SortedMarkdownRenderer) title(title string) error {
	_, err := fmt.Fprintln(r.Writer, "## "+strings.ToUpper(title))
	return err
}

func (r SortedMarkdownRenderer) groupBreak() error {
	const groupBreak = "\n\n"
	_, err := fmt.Fprint(r.Writer, groupBreak)
	return err
}

func (r SortedMarkdownRenderer) item(i api.Item) error {
	_, err := fmt.Fprintln(r.Writer, "- "+i.Name)
	return err
}
