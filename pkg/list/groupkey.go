package list

import (
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/golang/protobuf/proto"
)

type GroupKeys []*api.GroupKey

func (gks GroupKeys) Contains(k api.GroupKey) bool {
	for _, gk := range gks {
		if proto.Equal(gk, &k) {
			return true
		}
	}
	return false
}
