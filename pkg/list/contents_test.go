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

	t.Run("trims space before input", func(t *testing.T) {
		input := `	foo  `
		expected := api.Contents{
			Items: []string{"foo"},
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
			Refs: list.References{"foo"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple groups", func(t *testing.T) {
		input := "ref:foo\nref:bar"
		expected := api.Contents{
			Refs: list.References{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups", func(t *testing.T) {
		input := "foo\nref:foo\nbar\nref:bar"
		expected := api.Contents{
			Items: list.Items{"foo", "bar"},
			Refs:  list.References{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with whitespace", func(t *testing.T) {
		input := "\n  foo\n\tref:foo\nbar\nref:bar"
		expected := api.Contents{
			Items: list.Items{"foo", "bar"},
			Refs:  list.References{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with comment at start of line", func(t *testing.T) {
		input := "# some comment\nfoo\nref:foo\nbar\nref:bar"
		expected := api.Contents{
			Items: list.Items{"foo", "bar"},
			Refs:  list.References{"foo", "bar"},
		}
		actual, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("reject line starting with tag that isn't ref", func(t *testing.T) {
		input := "no:foo"
		_, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.EqualError(t, err, `processing line 1: "no:foo": unsupported tag prefix: "no"`)
	})

	t.Run("reject line with reference tag prefix but empty reference", func(t *testing.T) {
		input := "ref: "
		_, err := list.ParseContentsDefinition(strings.NewReader(input))
		assert.EqualError(t, err, `processing line 1: "ref:": empty reference`)
	})
}
