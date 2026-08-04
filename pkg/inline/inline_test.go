package inline_test

import (
	"testing"

	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/inline"
	"github.com/stretchr/testify/assert"
)

func TestSingleItemGroups(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, inline.SingleItemGroups(nil))
	})

	t.Run("no single-item groups leaves input unchanged", func(t *testing.T) {
		in := []api.Group{
			{Name: "camping", Contents: api.Contents{Refs: []string{"tent"}}},
			{Name: "tent", Contents: api.Contents{Items: []string{"Poles", "Flysheet"}}},
		}
		assert.Equal(t, in, inline.SingleItemGroups(in))
	})

	t.Run("referenced single-item group is inlined and dropped", func(t *testing.T) {
		in := []api.Group{
			{Name: "camping", Contents: api.Contents{Refs: []string{"light"}}},
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}}},
		}
		expected := []api.Group{
			{Name: "camping", Contents: api.Contents{Items: []string{"Light"}}},
		}
		assert.Equal(t, expected, inline.SingleItemGroups(in))
	})

	t.Run("single-item group referenced by two parents is inlined into both", func(t *testing.T) {
		in := []api.Group{
			{Name: "camping", Contents: api.Contents{Refs: []string{"light"}}},
			{Name: "cycling", Contents: api.Contents{Refs: []string{"light"}}},
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}}},
		}
		expected := []api.Group{
			{Name: "camping", Contents: api.Contents{Items: []string{"Light"}}},
			{Name: "cycling", Contents: api.Contents{Items: []string{"Light"}}},
		}
		assert.Equal(t, expected, inline.SingleItemGroups(in))
	})

	t.Run("unreferenced single-item group is left as-is", func(t *testing.T) {
		in := []api.Group{
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}}},
		}
		assert.Equal(t, in, inline.SingleItemGroups(in))
	})

	t.Run("group referencing two single-item groups inlines both in ref order", func(t *testing.T) {
		in := []api.Group{
			{Name: "bag", Contents: api.Contents{Refs: []string{"light", "spork"}}},
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}}},
			{Name: "spork", Contents: api.Contents{Items: []string{"Spork"}}},
		}
		expected := []api.Group{
			{Name: "bag", Contents: api.Contents{Items: []string{"Light", "Spork"}}},
		}
		assert.Equal(t, expected, inline.SingleItemGroups(in))
	})

	t.Run("only the single-item ref is inlined, other refs and items are kept", func(t *testing.T) {
		in := []api.Group{
			{Name: "trip", Contents: api.Contents{Refs: []string{"light", "camping"}, Items: []string{"Map"}}},
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}}},
			{Name: "camping", Contents: api.Contents{Items: []string{"Tent", "Poles"}}},
		}
		expected := []api.Group{
			{Name: "trip", Contents: api.Contents{Refs: []string{"camping"}, Items: []string{"Map", "Light"}}},
			{Name: "camping", Contents: api.Contents{Items: []string{"Tent", "Poles"}}},
		}
		assert.Equal(t, expected, inline.SingleItemGroups(in))
	})

	t.Run("requires on a surviving group is preserved", func(t *testing.T) {
		in := []api.Group{
			{Name: "kit", Contents: api.Contents{Refs: []string{"light"}, Items: []string{"Map"}, Requires: []string{"firstaid"}}},
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}}},
		}
		expected := []api.Group{
			{Name: "kit", Contents: api.Contents{Items: []string{"Map", "Light"}, Requires: []string{"firstaid"}}},
		}
		assert.Equal(t, expected, inline.SingleItemGroups(in))
	})

	t.Run("a single-item group with requires is not treated as single-item-only but still collapses", func(t *testing.T) {
		// The predicate intentionally ignores Requires (clients never see it), so a
		// referenced group with one item and a requirement still inlines.
		in := []api.Group{
			{Name: "camping", Contents: api.Contents{Refs: []string{"light"}}},
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}, Requires: []string{"battery"}}},
		}
		expected := []api.Group{
			{Name: "camping", Contents: api.Contents{Items: []string{"Light"}}},
		}
		assert.Equal(t, expected, inline.SingleItemGroups(in))
	})

	t.Run("does not mutate the caller's input slices", func(t *testing.T) {
		parentRefs := []string{"light"}
		in := []api.Group{
			{Name: "camping", Contents: api.Contents{Refs: parentRefs}},
			{Name: "light", Contents: api.Contents{Items: []string{"Light"}}},
		}
		_ = inline.SingleItemGroups(in)
		assert.Equal(t, []string{"light"}, parentRefs, "input refs should be untouched")
		assert.Equal(t, []string{"light"}, in[0].Contents.Refs)
	})
}
