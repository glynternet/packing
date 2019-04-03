package write

import (
	"fmt"
	"io"
	"strings"

	"github.com/glynternet/packing/pkg/list"
)

// TODO: There should be a Renderer here, maybe.
//	type Renderer struct {
//		io.Writer
//	}
//
// func (r *Renderer)RenderGroup(g list.Group) error {...}
// etc...

func Group(w io.Writer, g list.Group) {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		name = "Unnamed"
	}
	Title(w, name)
	for _, item := range g.Items {
		Item(w, item)
	}
}

func Title(w io.Writer, text string) {
	fmt.Fprintln(w, "## "+strings.ToUpper(text))
}

func GroupBreak(w io.Writer) {
	const groupBreak = "\n\n"
	fmt.Fprint(w, groupBreak)
}

func Item(w io.Writer, item string) {
	fmt.Fprintln(w, "- "+item)
}
