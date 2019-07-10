package load

import (
	"log"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

func Groups(keys list.GroupKeys, logger *log.Logger, cg ContentsDefinitionGetter) ([]list.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	loaded := make(map[string]list.ContentsDefinition)
	err := recursiveGroupsLoad(keys, logger, cg, loaded)
	var groups []list.Group
	for n, cs := range loaded {
		groups = append(groups, list.Group{
			Name: n, ContentsDefinition: cs,
		})
	}
	return groups, err
}

func recursiveGroupsLoad(keys list.GroupKeys, logger *log.Logger, cg ContentsDefinitionGetter, loaded map[string]list.ContentsDefinition) error {
	if len(keys) == 0 {
		return nil
	}
	var subgroupKeys list.GroupKeys
	for _, key := range keys {
		if _, ok := loaded[string(key)]; ok {
			// skip if exists
			continue
		}

		cs, err := cg.GetContentsDefinition(key)
		if err != nil {
			return errors.Wrapf(err, "getting group for key:%q", key)
		}

		if cs == nil {
			continue
		}

		if cs.GroupKeys.Contains(key) {
			return GroupSelfReferenceErr
		}

		loaded[string(key)] = *cs
		subgroupKeys = append(subgroupKeys, cs.GroupKeys...)
	}
	return recursiveGroupsLoad(subgroupKeys, logger, cg, loaded)
}

func AllGroups(logger *log.Logger, def list.ContentsDefinition, cg storage.ContentsDefinitionGetter) ([]list.Group, error) {
	groups, err := Groups(def.GroupKeys, logger, cg)
	if err != nil {
		return nil, errors.Wrap(err, "loading groups recursively")
	}
	if len(def.Items) > 0 {
		groups = append(groups, list.Group{
			Name: "Individual Items",
			ContentsDefinition: list.ContentsDefinition{
				Items: def.Items,
			},
		})
	}
	return groups, nil
}
