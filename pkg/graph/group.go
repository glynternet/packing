package graph

import (
	api "github.com/glynternet/packing/pkg/api/build"
)

type Group struct {
	api.Group
	ImportedBy []string
}

func (g Group) HasContents() bool {
	if g.HasItems() {
		return true
	}

	return len(g.Contents.GroupKeys) != 0
}

func (g Group) HasItems() bool {
	return len(g.Contents.Items) > 0
}
