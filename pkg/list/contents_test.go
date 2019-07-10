package list

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseContentsDefinition(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		input := ``
		expected := ContentsDefinition{}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single item", func(t *testing.T) {
		input := `foo`
		expected := ContentsDefinition{
			Items: Items{"foo"},
		}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple items", func(t *testing.T) {
		input := "foo\nbar"
		expected := ContentsDefinition{
			Items: Items{"foo", "bar"},
		}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single group", func(t *testing.T) {
		input := "group:foo"
		expected := ContentsDefinition{
			GroupKeys: GroupKeys{"foo"},
		}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple groups", func(t *testing.T) {
		input := "group:foo\ngroup:bar"
		expected := ContentsDefinition{
			GroupKeys: GroupKeys{"foo", "bar"},
		}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups", func(t *testing.T) {
		input := "foo\ngroup:foo\nbar\ngroup:bar"
		expected := ContentsDefinition{
			Items:     Items{"foo", "bar"},
			GroupKeys: GroupKeys{"foo", "bar"},
		}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with whitespace", func(t *testing.T) {
		input := "\n  foo\n\tgroup:foo\nbar\ngroup:bar"
		expected := ContentsDefinition{
			Items:     Items{"foo", "bar"},
			GroupKeys: GroupKeys{"foo", "bar"},
		}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with comment at start of line", func(t *testing.T) {
		input := "# some comment\nfoo\ngroup:foo\nbar\ngroup:bar"
		expected := ContentsDefinition{
			Items:     Items{"foo", "bar"},
			GroupKeys: GroupKeys{"foo", "bar"},
		}
		actual, err := ParseContentsDefinition(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
