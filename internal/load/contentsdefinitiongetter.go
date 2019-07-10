package load

import "github.com/glynternet/packing/pkg/list"

// ContentsDefinitionGetter gets the ContentsDefinition for a single group from a given key
type ContentsDefinitionGetter interface {
	GetContentsDefinition(list.GroupKey) (*list.ContentsDefinition, error)
}
