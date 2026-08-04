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

func TestPassthroughRefs(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, inline.PassthroughRefs(nil))
	})

	for _, tc := range []struct {
		name     string
		in       []api.Group
		expected []api.Group
	}{
		{
			// A group whose entire content is a single ref (e.g. "trip") would
			// itself be a passthrough, so the parent here carries an item to stay
			// a real group.
			name: "no passthrough groups leaves input unchanged",
			in: []api.Group{
				{Name: "trip", Contents: api.Contents{Refs: []string{"camping"}, Items: []string{"Map"}}},
				{Name: "camping", Contents: api.Contents{Items: []string{"Tent", "Poles"}}},
			},
			expected: []api.Group{
				{Name: "trip", Contents: api.Contents{Refs: []string{"camping"}, Items: []string{"Map"}}},
				{Name: "camping", Contents: api.Contents{Items: []string{"Tent", "Poles"}}},
			},
		},
		{
			// P carries an item so it is a real parent (not itself a passthrough)
			// and survives with its ref rewritten to the chain terminal.
			name: "chain to multi-item terminal rewrites parent and drops intermediates",
			in: []api.Group{
				{Name: "P", Contents: api.Contents{Refs: []string{"a"}, Items: []string{"Map"}}},
				{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
				{Name: "b", Contents: api.Contents{Refs: []string{"c"}}},
				{Name: "c", Contents: api.Contents{Items: []string{"Tent", "Poles"}}},
			},
			expected: []api.Group{
				{Name: "P", Contents: api.Contents{Refs: []string{"c"}, Items: []string{"Map"}}},
				{Name: "c", Contents: api.Contents{Items: []string{"Tent", "Poles"}}},
			},
		},
		{
			name: "cycle is left intact with refs unchanged and nothing dropped",
			in: []api.Group{
				{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
				{Name: "b", Contents: api.Contents{Refs: []string{"a"}}},
			},
			expected: []api.Group{
				{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
				{Name: "b", Contents: api.Contents{Refs: []string{"a"}}},
			},
		},
		{
			name: "two heads sharing a terminal are deduped to a single ref and both dropped",
			in: []api.Group{
				{Name: "P", Contents: api.Contents{Refs: []string{"h1", "h2"}}},
				{Name: "h1", Contents: api.Contents{Refs: []string{"t"}}},
				{Name: "h2", Contents: api.Contents{Refs: []string{"t"}}},
				{Name: "t", Contents: api.Contents{Items: []string{"Rope", "Carabiner"}}},
			},
			expected: []api.Group{
				{Name: "P", Contents: api.Contents{Refs: []string{"t"}}},
				{Name: "t", Contents: api.Contents{Items: []string{"Rope", "Carabiner"}}},
			},
		},
		{
			name: "passthrough mixed with a normal ref rewrites only the passthrough and keeps the group",
			in: []api.Group{
				{Name: "g", Contents: api.Contents{Refs: []string{"x", "y"}}},
				{Name: "x", Contents: api.Contents{Refs: []string{"t"}}},
				{Name: "y", Contents: api.Contents{Items: []string{"Yo"}}},
				{Name: "t", Contents: api.Contents{Items: []string{"Tee", "Two"}}},
			},
			expected: []api.Group{
				{Name: "g", Contents: api.Contents{Refs: []string{"t", "y"}}},
				{Name: "y", Contents: api.Contents{Items: []string{"Yo"}}},
				{Name: "t", Contents: api.Contents{Items: []string{"Tee", "Two"}}},
			},
		},
		{
			// A top-level passthrough referenced by nothing (only the seed, which
			// is not part of the group list) is still dropped; its terminal is
			// left as a standalone orphan group.
			name: "top-level passthrough orphan is dropped and its terminal remains",
			in: []api.Group{
				{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
				{Name: "b", Contents: api.Contents{Items: []string{"Bee", "Buzz"}}},
			},
			expected: []api.Group{
				{Name: "b", Contents: api.Contents{Items: []string{"Bee", "Buzz"}}},
			},
		},
		{
			name: "cycle head referenced by a parent keeps the ref to the cycle",
			in: []api.Group{
				{Name: "P", Contents: api.Contents{Refs: []string{"a"}}},
				{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
				{Name: "b", Contents: api.Contents{Refs: []string{"a"}}},
			},
			expected: []api.Group{
				{Name: "P", Contents: api.Contents{Refs: []string{"a"}}},
				{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
				{Name: "b", Contents: api.Contents{Refs: []string{"a"}}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, inline.PassthroughRefs(tc.in))
		})
	}

	t.Run("chain to single-item terminal composes with SingleItemGroups", func(t *testing.T) {
		// P carries an item so it survives PassthroughRefs; the chain terminal c
		// is a single-item group, so SingleItemGroups then inlines it into P.
		in := []api.Group{
			{Name: "P", Contents: api.Contents{Refs: []string{"a"}, Items: []string{"Map"}}},
			{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
			{Name: "b", Contents: api.Contents{Refs: []string{"c"}}},
			{Name: "c", Contents: api.Contents{Items: []string{"Light"}}},
		}
		expected := []api.Group{
			{Name: "P", Contents: api.Contents{Items: []string{"Map", "Light"}}},
		}
		assert.Equal(t, expected, inline.SingleItemGroups(inline.PassthroughRefs(in)))
	})

	t.Run("does not mutate the caller's input slices", func(t *testing.T) {
		parentRefs := []string{"a"}
		in := []api.Group{
			{Name: "P", Contents: api.Contents{Refs: parentRefs}},
			{Name: "a", Contents: api.Contents{Refs: []string{"b"}}},
			{Name: "b", Contents: api.Contents{Items: []string{"Bee"}}},
		}
		_ = inline.PassthroughRefs(in)
		assert.Equal(t, []string{"a"}, parentRefs, "input refs should be untouched")
		assert.Equal(t, []string{"a"}, in[0].Contents.Refs)
	})
}
