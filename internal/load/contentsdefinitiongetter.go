package load

import (
	"github.com/glynternet/packing/pkg/api"
)

// ContentsDefinitionGetter gets the ContentsDefinition for a single reference
type ContentsDefinitionGetter interface {
	GetContentsDefinition(string) (api.Contents, error)
}
