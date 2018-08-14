package write

import (
	"io"
	"strings"
	"fmt"
	"github.com/glynternet/packing/pkg/list"
)

func Group(w io.Writer, g list.Group) {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		name = "Unnamed"
	}
	Title(w, name)
	for _, item := range g.Items {
		fmt.Fprintln(w, item)
	}
}

func Title(w io.Writer, text string) {
	fmt.Fprintln(w, strings.ToUpper(text))
}

func GroupBreak(w io.Writer) {
	const groupBreak = "\n\n"
	fmt.Fprint(w, groupBreak)
}
