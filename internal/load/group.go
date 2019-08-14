package load

import (
	"log"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

func Groups(keys []*api.GroupKey, logger *log.Logger, cg ContentsDefinitionGetter) ([]list.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	loaded := make(map[string]api.ContentsDefinition)
	err := recursiveGroupsLoad(keys, logger, cg, loaded)
	var groups []list.Group
	for n, cs := range loaded {
		groups = append(groups, list.Group{
			Name: n, ContentsDefinition: cs,
		})
	}
	return groups, err
}

func recursiveGroupsLoad(keys list.GroupKeys, logger *log.Logger, cg ContentsDefinitionGetter, loaded map[string]api.ContentsDefinition) error {
	if len(keys) == 0 {
		return nil
	}
	var subgroupKeys list.GroupKeys
	for _, key := range keys {
		if _, ok := loaded[key.Key]; ok {
			// skip if exists
			continue
		}

		cs, err := cg.GetContentsDefinition(*key)
		if err != nil {
			return errors.Wrapf(err, "getting group for key:%q", key)
		}

		if cs == nil {
			continue
		}

		if cs.GroupKeys != nil && list.GroupKeys(cs.GroupKeys).Contains(*key) {
			return GroupSelfReferenceErr
		}

		loaded[key.Key] = *cs
		if cs.GroupKeys == nil {
			continue
		}
		subgroupKeys = append(subgroupKeys, cs.GroupKeys...)
	}
	return recursiveGroupsLoad(subgroupKeys, logger, cg, loaded)
}

func AllGroups(logger *log.Logger, def api.ContentsDefinition, cg storage.ContentsDefinitionGetter) ([]list.Group, error) {
	groups, err := Groups(def.GroupKeys, logger, cg)
	if err != nil {
		return nil, errors.Wrap(err, "loading groups recursively")
	}
	if len(def.Items) > 0 {
		groups = append(groups, list.Group{
			Name: "Individual Items",
			ContentsDefinition: api.ContentsDefinition{
				Items: def.Items,
			},
		})
	}
	return groups, nil
}
