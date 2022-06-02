package graph

import "github.com/glynternet/packing/pkg/api"

type Group struct {
	Group      api.Group
	ImportedBy []string
}

func (g Group) HasContents() bool {
	if len(g.Group.GetContents().GetItems()) != 0 {
		return true
	}

	return len(g.Group.GetContents().GetReferences()) != 0
}
