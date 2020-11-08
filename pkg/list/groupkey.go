package list

import (
	api "github.com/glynternet/packing/pkg/api/build"
	"google.golang.org/protobuf/proto"
)

// GroupKeys is a set of api.GroupKey
type GroupKeys []*api.GroupKey

// Contains returns true if the GroupKeys contain the given api.GroupKey
func (gks GroupKeys) Contains(k *api.GroupKey) bool {
	for _, gk := range gks {
		if proto.Equal(gk, k) {
			return true
		}
	}
	return false
}

func (gks GroupKeys) Strings() []string {
	var ss []string
	for _, gk := range gks {
		ss = append(ss, gk.GetKey())
	}
	return ss
}
