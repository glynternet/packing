package write

import (
	"fmt"
	"io"
	"strings"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

// TODO(glynternet): There should be a Renderer here, maybe.
//   definitely, there should be, for example, a markdown renderer that does
//   things like escaping underscores to make them valid markdown
//	type Renderer struct {
//		io.Writer
//	}
//
// func (r *Renderer)RenderGroup(g list.Group) error {...}
// etc...

// Group writes a list.Group to the given io.Writer
func Group(w io.Writer, g list.Group) error {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		name = "Unnamed"
	}
	if err := Title(w, name); err != nil {
		return errors.Wrapf(err, "writing Title %q to writer", name)
	}
	for _, item := range g.Items {
		if err := Item(w, *item); err != nil {
			return errors.Wrapf(err, "writing Item %q to writer", *item)
		}
	}
	return nil
}

// Title writes the given string as a title to the given io.Writer
func Title(w io.Writer, title string) error {
	_, err := fmt.Fprintln(w, "## "+strings.ToUpper(title))
	return err
}

// GroupBreak writes a gap to be used between each list.Group to the io.Writer
func GroupBreak(w io.Writer) error {
	const groupBreak = "\n\n"
	_, err := fmt.Fprint(w, groupBreak)
	return err
}

// Item writes an api.Item to the given io.Writer
func Item(w io.Writer, i api.Item) error {
	_, err := fmt.Fprintln(w, "- "+i.Name)
	return err
}
