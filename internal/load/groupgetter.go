package load

import "github.com/glynternet/packing/pkg/list"

type GroupGetter interface {
	GetGroup(string) (list.Group, error)
}
