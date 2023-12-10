package load

import (
	"github.com/glynternet/packing/pkg/api"
)

// ContentsDefinitionGetter gets the ContentsDefinition for a single group from a given key
type ContentsDefinitionGetter interface {
	GetContentsDefinition(string) (api.Contents, error)
}
