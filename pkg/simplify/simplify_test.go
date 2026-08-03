package simplify_test

import (
	"testing"

	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/simplify"
	"github.com/stretchr/testify/assert"
)

func group(name string, refs, items []string) api.Group {
	return api.Group{Name: name, Contents: api.Contents{Refs: refs, Items: items}}
}

func TestSimplify(t *testing.T) {
	t.Run("empty seed removes nothing", func(t *testing.T) {
		res := simplify.Simplify(api.Contents{}, nil)
		assert.Empty(t, res.RemovedRefs)
		assert.Empty(t, res.RemovedItems)
	})

	t.Run("ref reachable via another ref is removed", func(t *testing.T) {
		seed := api.Contents{Refs: []string{"camping", "tent"}}
		groups := []api.Group{
			group("camping", []string{"tent"}, nil),
			group("tent", nil, []string{"Poles"}),
		}
		res := simplify.Simplify(seed, groups)
		assert.Equal(t, []string{"tent"}, res.RemovedRefs)
		assert.Equal(t, []string{"camping"}, res.Refs)
	})

	t.Run("unrelated sibling refs are kept", func(t *testing.T) {
		seed := api.Contents{Refs: []string{"a", "b"}}
		groups := []api.Group{
			group("a", nil, []string{"Apple"}),
			group("b", nil, []string{"Banana"}),
		}
		res := simplify.Simplify(seed, groups)
		assert.Empty(t, res.RemovedRefs)
		assert.Equal(t, []string{"a", "b"}, res.Refs)
	})

	t.Run("mutual cycle keeps exactly one ref", func(t *testing.T) {
		seed := api.Contents{Refs: []string{"a", "b"}}
		groups := []api.Group{
			group("a", []string{"b"}, nil),
			group("b", []string{"a"}, nil),
		}
		res := simplify.Simplify(seed, groups)
		assert.Equal(t, []string{"a"}, res.RemovedRefs)
		assert.Equal(t, []string{"b"}, res.Refs)
	})

	t.Run("item present in a reachable group is removed", func(t *testing.T) {
		seed := api.Contents{Refs: []string{"tent"}, Items: []string{"Poles", "Sunscreen"}}
		groups := []api.Group{group("tent", nil, []string{"Poles"})}
		res := simplify.Simplify(seed, groups)
		assert.Equal(t, []string{"Poles"}, res.RemovedItems)
		assert.Equal(t, []string{"Sunscreen"}, res.Items)
	})

	t.Run("item nested deep in the ref tree is removed", func(t *testing.T) {
		seed := api.Contents{Refs: []string{"camping"}, Items: []string{"Poles"}}
		groups := []api.Group{
			group("camping", []string{"tent"}, nil),
			group("tent", nil, []string{"Poles"}),
		}
		res := simplify.Simplify(seed, groups)
		assert.Equal(t, []string{"Poles"}, res.RemovedItems)
	})

	t.Run("item absent from all groups is kept", func(t *testing.T) {
		seed := api.Contents{Refs: []string{"tent"}, Items: []string{"Passport"}}
		groups := []api.Group{group("tent", nil, []string{"Poles"})}
		res := simplify.Simplify(seed, groups)
		assert.Empty(t, res.RemovedItems)
		assert.Equal(t, []string{"Passport"}, res.Items)
	})

	t.Run("duplicate refs are not mis-removed", func(t *testing.T) {
		seed := api.Contents{Refs: []string{"a", "a"}}
		groups := []api.Group{group("a", nil, []string{"Apple"})}
		res := simplify.Simplify(seed, groups)
		assert.Empty(t, res.RemovedRefs)
		assert.Equal(t, []string{"a"}, res.Refs)
	})
}

func TestFilter(t *testing.T) {
	t.Run("drops removed ref lines, keeps comments/blanks/order", func(t *testing.T) {
		text := "# my trip\nref: camping\nref: tent\n\nSunscreen\n"
		got := simplify.Filter(text, []string{"tent"}, nil)
		assert.Equal(t, "# my trip\nref: camping\n\nSunscreen\n", got)
	})

	t.Run("drops removed item lines", func(t *testing.T) {
		text := "ref: tent\nPoles\nSunscreen\n"
		got := simplify.Filter(text, nil, []string{"Poles"})
		assert.Equal(t, "ref: tent\nSunscreen\n", got)
	})

	t.Run("keeps req lines and unrelated refs/items", func(t *testing.T) {
		text := "req: passport\nref: tent\nPoles\n"
		got := simplify.Filter(text, []string{"other"}, []string{"other"})
		assert.Equal(t, text, got)
	})

	t.Run("classifies ref value ignoring inline comment and spacing", func(t *testing.T) {
		text := "ref:  tent   # camping tent\n"
		got := simplify.Filter(text, []string{"tent"}, nil)
		assert.Equal(t, "", got)
	})

	t.Run("item value ignores inline comment", func(t *testing.T) {
		text := "Shave before going # reminder\nPoles\n"
		got := simplify.Filter(text, nil, []string{"Shave before going"})
		assert.Equal(t, "Poles\n", got)
	})

	t.Run("nothing removed round-trips exactly", func(t *testing.T) {
		text := "# trip\nref: tent\n\nPoles\n"
		assert.Equal(t, text, simplify.Filter(text, nil, nil))
	})
}
