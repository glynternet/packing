package list

import (
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/golang/protobuf/proto"
)

type Item string

func ExtractItem(item api.Item) Item {
	return Item(item.Name)
}

type Items []*api.Item

func (is Items) Contains(i *api.Item) bool {
	for _, ii := range is {
		if proto.Equal((*api.Item)(ii), (*api.Item)(i)) {
			return true
		}
	}
	return false
}
