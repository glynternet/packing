package storage_test

import (
	"strings"
	"testing"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestLoadContents(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		input := ``
		expected := list.Contents{}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single item", func(t *testing.T) {
		input := `foo`
		expected := list.Contents{
			Items: list.Items{"foo"},
		}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple items", func(t *testing.T) {
		input := "foo\nbar"
		expected := list.Contents{
			Items: list.Items{"foo", "bar"},
		}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single group", func(t *testing.T) {
		input := "group:foo"
		expected := list.Contents{
			GroupKeys: list.GroupKeys{"foo"},
		}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("multiple groups", func(t *testing.T) {
		input := "group:foo\ngroup:bar"
		expected := list.Contents{
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups", func(t *testing.T) {
		input := "foo\ngroup:foo\nbar\ngroup:bar"
		expected := list.Contents{
			Items:     list.Items{"foo", "bar"},
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with whitespace", func(t *testing.T) {
		input := "\n  foo\n\tgroup:foo\nbar\ngroup:bar"
		expected := list.Contents{
			Items:     list.Items{"foo", "bar"},
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("items and groups with comment at start of line", func(t *testing.T) {
		input := "# some comment\nfoo\ngroup:foo\nbar\ngroup:bar"
		expected := list.Contents{
			Items:     list.Items{"foo", "bar"},
			GroupKeys: list.GroupKeys{"foo", "bar"},
		}
		actual, err := storage.LoadContents(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
