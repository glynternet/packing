package graph

import "github.com/glynternet/packing/pkg/api"

type Group struct {
	Group      api.Group
	ImportedBy []string
}

func (g Group) HasContents() bool {
	return g.HasItems() || len(g.Group.Contents.Refs) > 0
}

func (g Group) HasItems() bool {
	return len(g.Group.Contents.Items) > 0
}
