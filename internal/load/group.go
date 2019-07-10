package load

import (
	"log"

	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

func Groups(keys []string, logger *log.Logger, cg ContentsGetter) (map[string]list.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	groups := make(map[string]list.Group)
	err := recursiveGroupsLoad(keys, logger, cg, groups)
	return groups, err
}

func recursiveGroupsLoad(keys []string, logger *log.Logger, cg ContentsGetter, groups map[string]list.Group) error {
	if len(keys) == 0 {
		return nil
	}
	var subgroupKeys []string
	for _, key := range keys {
		if _, ok := groups[key]; ok {
			// skip if exists
			continue
		}

		cs, err := cg.GetContents(key)
		if err != nil {
			return errors.Wrapf(err, "getting group for key:%q", key)
		}

		if len(cs.Items) > 0 {
			groups[key] = list.Group{
				Name: key,
				Contents: list.Contents{
					Items: cs.Items,
				},
			}
		}
		subgroupKeys = append(subgroupKeys, cs.GroupKeys...)
	}
	return recursiveGroupsLoad(subgroupKeys, logger, cg, groups)
}
