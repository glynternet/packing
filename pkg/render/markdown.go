package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/glynternet/packing/pkg/graph"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

// SortedMarkdownRenderer renders a graph to its writer sorted by group name
type SortedMarkdownRenderer struct {
	IncludeEmptyParentGroups bool
	IncludeGroupReferences   bool
}

// Render renders a graph to the SortedMarkdownRenderer's writer sorted by group name
func (r SortedMarkdownRenderer) Render(w io.Writer, gs []graph.Group) error {
	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Group.Name < gs[j].Group.Name
	})

	for _, g := range gs {
		if !g.HasContents() {
			continue
		}
		if !g.HasItems() && !r.IncludeEmptyParentGroups {
			continue
		}
		if err := r.group(w, g); err != nil {
			return errors.Wrapf(err, "writing Group %v to writer", g)
		}
		if err := groupBreak(w); err != nil {
			return errors.Wrapf(err, "writing GroupBreak %v to writer", g)
		}
	}
	return nil
}

func (r SortedMarkdownRenderer) group(w io.Writer, g graph.Group) error {
	name := strings.TrimSpace(g.Group.Name)
	if name == "" {
		name = "Unnamed"
	}
	if err := title(w, name); err != nil {
		return errors.Wrapf(err, "writing Title %q to writer", name)
	}
	if r.IncludeGroupReferences {
		if err := includedIns(w, g.ImportedBy); err != nil {
			return errors.Wrapf(err, "writing ImportedBy %q to writer", g.ImportedBy)
		}
	}
	if r.IncludeGroupReferences {
		includesGroups := list.References(g.Group.Contents.Refs)
		if err := includes(w, includesGroups); err != nil {
			return errors.Wrapf(err, "writing includes %q to writer", includesGroups)
		}
	}
	for _, contentItem := range g.Group.Contents.Items {
		if err := item(w, contentItem); err != nil {
			return errors.Wrapf(err, "writing Item %v to writer", contentItem)
		}
	}
	return nil
}

func title(w io.Writer, title string) error {
	_, err := fmt.Fprintln(w, "## "+escaped(strings.ToUpper(title)))
	return err
}

func groupBreak(w io.Writer) error {
	const groupBreak = "\n\n"
	_, err := fmt.Fprint(w, groupBreak)
	return err
}

func item(w io.Writer, name string) error {
	_, err := fmt.Fprintln(w, "- "+escaped(name))
	return err
}

func includes(w io.Writer, is []string) error {
	if len(is) == 0 {
		return nil
	}
	sorted := make([]string, len(is))
	copy(sorted, is)
	sort.Strings(sorted)
	var anchors []string
	for _, group := range sorted {
		anchors = append(anchors, anchor(escaped(group), sanitisedHeaderURLID(group)))
	}
	_, err := fmt.Fprintf(w, "_Includes groups: %s_  \n\n", strings.Join(anchors, ", "))
	return err
}

func includedIns(w io.Writer, is []string) error {
	if len(is) == 0 {
		return nil
	}
	sorted := make([]string, len(is))
	copy(sorted, is)
	sort.Strings(sorted)
	var anchors []string
	for _, group := range sorted {
		anchors = append(anchors, anchor(escaped(group), sanitisedHeaderURLID(group)))
	}
	_, err := fmt.Fprintf(w, "_Included in: %s_  \n\n", strings.Join(anchors, ", "))
	return err
}

func anchor(text, url string) string {
	return fmt.Sprintf("[%s](#%s)", text, url)
}

func escaped(in string) string {
	return strings.ReplaceAll(in, "_", `\_`)
}

// this is annoyingly how the markdown to HTML renderer renders heading IDs for links
func sanitisedHeaderURLID(v string) string {
	return strings.ReplaceAll(v, "_", `-`)
}
