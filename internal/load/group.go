package load

import (
	"log"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

func Groups(keys []string, logger *log.Logger, cg storage.GroupGetter) (map[string]list.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	groups := make(map[string]list.Group)
	err := recursiveGroupsLoad(keys, logger, cg, groups)
	return groups, err
}

func recursiveGroupsLoad(keys []string, logger *log.Logger, cg storage.GroupGetter, groups map[string]list.Group) error {
	if len(keys) == 0 {
		return nil
	}
	var subgroupKeys []string
	for _, key := range keys {
		if _, ok := groups[key]; ok {
			// skip if exists
			continue
		}

		g, err := cg.GetGroup(key)
		if err != nil {
			return errors.Wrapf(err, "getting contents for key:%v", key)
		}

		if len(g.Items) > 0 {
			groups[key] = list.Group{
				Name:  key,
				Items: g.Items,
			}
		}
		subgroupKeys = append(subgroupKeys, g.GroupKeys...)
	}
	return recursiveGroupsLoad(subgroupKeys, logger, cg, groups)
}
