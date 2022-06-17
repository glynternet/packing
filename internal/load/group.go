package load

import (
	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
)

// Loader is used to load refs and content
type Loader struct {
	ContentsDefinitionGetter
}

// Groups returns all list.Group for the given Refs, using the given ContentsDefinitionGetter
func (l Loader) Groups(logger log.Logger, refs list.References) ([]api.Group, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	loaded := make(map[string]api.Contents)
	err := recursiveLoad(refs, logger, l.ContentsDefinitionGetter, loaded)

	var groups []api.Group
	for n, cs := range loaded {
		groups = append(groups, api.Group{
			Name: n, Contents: cs,
		})
	}
	return groups, err
}

func recursiveLoad(
	refs list.References,
	logger log.Logger, cg ContentsDefinitionGetter,
	loaded map[string]api.Contents) error {
	if len(refs) == 0 {
		return nil
	}
	var childReferences list.References
	for _, ref := range refs {
		if _, ok := loaded[ref]; ok {
			// skip if already loaded
			continue
		}

		cs, err := cg.GetContentsDefinition(ref)
		if err != nil {
			return errors.Wrapf(err, "getting contents for ref:%q", ref)
		}

		if list.References(cs.Refs).Contains(ref) {
			return SelfReferenceError(ref)
		}

		// TODO: handler getting requirements here

		childReferences = append(childReferences, cs.Refs...)
		loaded[ref] = cs
	}
	return recursiveLoad(childReferences, logger, cg, loaded)
}

// AllGroups gets all Groups from the seed ContentsDefinition and recursively descending into all of those Groups
// until the whole tree has been found.
func (l *Loader) AllGroups(logger log.Logger, seed api.Contents) ([]api.Group, error) {
	groups, err := l.Groups(logger, seed.Refs)
	if err != nil {
		return nil, errors.Wrap(err, "loading refs recursively")
	}
	if len(seed.Items) > 0 {
		groups = append(groups, api.Group{
			Name:     "Individual Items",
			Contents: api.Contents{Items: seed.Items},
		})
	}
	return groups, nil
}
