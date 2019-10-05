package load

import (
	"log"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

type Loader struct {
	ContentsDefinitionGetter
}

// Groups returns all of the list.Group for the given GroupKeys, using the given ContentsDefinitionGetter
func (l Loader) Groups(logger *log.Logger, keys list.GroupKeys) ([]api.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	loaded := make(map[string]api.ContentsDefinition)
	err := recursiveGroupsLoad(keys, logger, l.ContentsDefinitionGetter, loaded)
	var groups []api.Group
	for n, cs := range loaded {
		csCopy := cs
		groups = append(groups, api.Group{
			Name: n, Contents: &csCopy,
		})
	}
	return groups, err
}

func recursiveGroupsLoad(
	keys list.GroupKeys,
	logger *log.Logger, cg ContentsDefinitionGetter,
	loaded map[string]api.ContentsDefinition) error {
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

// AllGroups gets all of the Groups from the seed ContentsDefinition and recursively descending into all of those Groups
// until the whole tree has been found.
func (l *Loader) AllGroups(logger *log.Logger, seed api.ContentsDefinition) (
	[]api.Group, error,
) {
	groups, err := l.Groups(logger, seed.GroupKeys)
	if err != nil {
		return nil, errors.Wrap(err, "loading groups recursively")
	}
	if len(seed.Items) > 0 {
		groups = append(groups, api.Group{
			Name: "Individual Items",
			Contents: &api.ContentsDefinition{
				Items: seed.Items,
			},
		})
	}
	return groups, nil
}
