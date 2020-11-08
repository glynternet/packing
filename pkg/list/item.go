package list

import (
	api "github.com/glynternet/packing/pkg/api/build"
	"google.golang.org/protobuf/proto"
)

// Items is a set of api.Item
type Items []*api.Item

// Contains returns true if the Items contains the given api.Item
func (is Items) Contains(i *api.Item) bool {
	for _, ii := range is {
		if proto.Equal(ii, i) {
			return true
		}
	}
	return false
}
