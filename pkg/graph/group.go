package graph

import (
	api "github.com/glynternet/packing/pkg/api/build"
)

type Group struct {
	api.Group
	ImportedBy []string
}
