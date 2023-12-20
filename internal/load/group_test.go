package load_test

import (
	"bytes"
	"testing"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGroups(t *testing.T) {
	t.Run("empty keys", func(t *testing.T) {
		actual, err := load.Loader{}.Groups(nil, list.GroupKeys{})
		assert.NoError(t, err)
		assert.Nil(t, actual)
	})

	logger := log.NewLogger(&bytes.Buffer{})

	t.Run("single key empty returns contents", func(t *testing.T) {
		keys := list.GroupKeys{"foo"}
		store := mockContentsGetter{}
		expected := []api.Group{{Name: "foo"}}
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single key error in getter", func(t *testing.T) {
		keys := list.GroupKeys{"foo"}
		expectedErr := errors.New("test error")
		store := mockContentsGetter{error: expectedErr}
		var expected []api.Group
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.Equal(t, expectedErr, errors.Cause(err))
		assert.Equal(t, expected, actual)
	})

	t.Run("single key exists", func(t *testing.T) {
		keys := list.GroupKeys{"foo"}
		store := mockContentsGetter{
			groups: map[string]api.Contents{
				"foo": {Items: list.Items{"fooItem"}},
				"bar": {Items: list.Items{"barItem"}},
			},
		}
		expected := []api.Group{{
			Name:     "foo",
			Contents: api.Contents{Items: list.Items{"fooItem"}}},
		}
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single key containing self group reference", func(t *testing.T) {
		// currently completes but will probably cause a bug when changing the way that groups are loaded
		keys := list.GroupKeys{"foo"}
		store := mockContentsGetter{
			groups: map[string]api.Contents{
				"foo": {GroupKeys: list.GroupKeys{"foo"}},
			},
		}
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.Equal(t, load.SelfReferenceError("foo"), err)
		assert.Nil(t, actual)
	})
}

type mockContentsGetter struct {
	groups map[string]api.Contents
	error
}

func (mgg mockContentsGetter) GetContentsDefinition(key string) (api.Contents, error) {
	g := mgg.groups[key]
	return g, mgg.error
}
