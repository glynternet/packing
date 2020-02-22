package graph

import (
	api "github.com/glynternet/packing/pkg/api/build"
)

type Group struct {
	api.Group
	ImportedBy []string
}

func (g Group) HasContents() bool {
	if len(g.Contents.Items) != 0 {
		return true
	}

	return len(g.Contents.GroupKeys) != 0
}
