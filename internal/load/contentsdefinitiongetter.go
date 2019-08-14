package load

import (
	api "github.com/glynternet/packing/pkg/api/build"
)

// ContentsDefinitionGetter gets the ContentsDefinition for a single group from a given key
type ContentsDefinitionGetter interface {
	GetContentsDefinition(api.GroupKey) (*api.ContentsDefinition, error)
}
