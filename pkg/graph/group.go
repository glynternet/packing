package graph

import (
	api "github.com/glynternet/packing/pkg/api/build"
)

type Group struct {
	Group      *api.Group
	ImportedBy []string
}

func (g Group) HasContents() bool {
	if len(g.Group.Contents.Items) != 0 {
		return true
	}

	return len(g.Group.Contents.GroupKeys) != 0
}
