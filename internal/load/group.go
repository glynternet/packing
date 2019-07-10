package load

import (
	"log"

	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

func Groups(keys []string, logger *log.Logger, cg ContentsGetter) ([]list.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	loaded := make(map[string]list.Contents)
	err := recursiveGroupsLoad(keys, logger, cg, loaded)
	var groups []list.Group
	for n, cs := range loaded {
		groups = append(groups, list.Group{
			Name: n, Contents: cs,
		})
	}
	return groups, err
}

func recursiveGroupsLoad(keys []string, logger *log.Logger, cg ContentsGetter, loaded map[string]list.Contents) error {
	if len(keys) == 0 {
		return nil
	}
	var subgroupKeys []string
	for _, key := range keys {
		if _, ok := loaded[key]; ok {
			// skip if exists
			continue
		}

		cs, err := cg.GetContents(key)
		if err != nil {
			return errors.Wrapf(err, "getting group for key:%q", key)
		}

		if len(cs.Items) > 0 {
			loaded[key] = list.Contents{
				Items: cs.Items,
			}
		}
		subgroupKeys = append(subgroupKeys, cs.GroupKeys...)
	}
	return recursiveGroupsLoad(subgroupKeys, logger, cg, loaded)
}
