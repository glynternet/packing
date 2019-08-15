package list

import api "github.com/glynternet/packing/pkg/api/build"

// Group is a named set of contents
type Group struct {
	Name string
	api.ContentsDefinition
}
