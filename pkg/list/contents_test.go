package list_test

import (
	"strings"
	"testing"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/list"
	"github.com/stretchr/testify/assert"
)

func TestParseContentsDefinition(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		input := ``
		expected := api.ContentsDefinition{}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single item", func(t *testing.T) {
		input := `foo`
		expected := api.ContentsDefinition{
			Items: list.Items{{Name: "foo"}},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple items", func(t *testing.T) {
		input := "foo\nbar"
		expected := api.ContentsDefinition{
			Items: list.Items{{Name: "foo"}, {Name: "bar"}},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single group", func(t *testing.T) {
		input := "group:foo"
		expected := api.ContentsDefinition{
			GroupKeys: list.GroupKeys{{Key: "foo"}},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple groups", func(t *testing.T) {
		input := "group:foo\ngroup:bar"
		expected := api.ContentsDefinition{
			GroupKeys: list.GroupKeys{{Key: "foo"}, {Key: "bar"}},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups", func(t *testing.T) {
		input := "foo\ngroup:foo\nbar\ngroup:bar"
		expected := api.ContentsDefinition{
			Items:     list.Items{{Name: "foo"}, {Name: "bar"}},
			GroupKeys: list.GroupKeys{{Key: "foo"}, {Key: "bar"}},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with whitespace", func(t *testing.T) {
		input := "\n  foo\n\tgroup:foo\nbar\ngroup:bar"
		expected := api.ContentsDefinition{
			Items:     list.Items{{Name: "foo"}, {Name: "bar"}},
			GroupKeys: list.GroupKeys{{Key: "foo"}, {Key: "bar"}},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with comment at start of line", func(t *testing.T) {
		input := "# some comment\nfoo\ngroup:foo\nbar\ngroup:bar"
		expected := api.ContentsDefinition{
			Items:     list.Items{{Name: "foo"}, {Name: "bar"}},
			GroupKeys: list.GroupKeys{{Key: "foo"}, {Key: "bar"}},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
