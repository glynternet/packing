package list_test

import (
	"strings"
	"testing"

	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/list"
	"github.com/stretchr/testify/assert"
)

func TestParseContentsDefinition(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		input := ``
		expected := api.Contents{}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single item", func(t *testing.T) {
		input := `foo`
		expected := api.Contents{
			Items: list.Items{"foo"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple items", func(t *testing.T) {
		input := "foo\nbar"
		expected := api.Contents{
			Items: list.Items{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single group", func(t *testing.T) {
		input := "ref:foo"
		expected := api.Contents{
			GroupKeys: list.GroupKeys{"foo"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple groups", func(t *testing.T) {
		input := "ref:foo\nref:bar"
		expected := api.Contents{
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups", func(t *testing.T) {
		input := "foo\nref:foo\nbar\nref:bar"
		expected := api.Contents{
			Items:     list.Items{"foo", "bar"},
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with whitespace", func(t *testing.T) {
		input := "\n  foo\n\tref:foo\nbar\nref:bar"
		expected := api.Contents{
			Items:     list.Items{"foo", "bar"},
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with comment at start of line", func(t *testing.T) {
		input := "# some comment\nfoo\nref:foo\nbar\nref:bar"
		expected := api.Contents{
			Items:     list.Items{"foo", "bar"},
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
