package load_test

import (
	"testing"

	"github.com/glynternet/packing/internal/load"
	api "github.com/glynternet/packing/pkg/api/build"
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

	logger := log.NewLogger()

	t.Run("single key missing in getter", func(t *testing.T) {
		keys := list.GroupKeys{{Key: "foo"}}
		store := mockContentsGetter{}
		var expected []*api.Group
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single key error in getter", func(t *testing.T) {
		keys := list.GroupKeys{{Key: "foo"}}
		expectedErr := errors.New("test error")
		store := mockContentsGetter{error: expectedErr}
		var expected []*api.Group
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.Equal(t, expectedErr, errors.Cause(err))
		assert.Equal(t, expected, actual)
	})

	t.Run("single key exists", func(t *testing.T) {
		keys := list.GroupKeys{{Key: "foo"}}
		store := mockContentsGetter{
			groups: map[string]api.ContentsDefinition{
				"foo": {Items: list.Items{{Name: "fooItem"}}},
				"bar": {Items: list.Items{{Name: "barItem"}}},
			},
		}
		expected := []*api.Group{{
			Name:     "foo",
			Contents: &api.ContentsDefinition{Items: list.Items{{Name: "fooItem"}}}},
		}
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("single key containing self group reference", func(t *testing.T) {
		// currently completes but will probably cause a bug when changing the way that groups are loaded
		keys := list.GroupKeys{{Key: "foo"}}
		store := mockContentsGetter{
			groups: map[string]api.ContentsDefinition{
				"foo": {GroupKeys: list.GroupKeys{{Key: "foo"}}},
			},
		}
		actual, err := load.Loader{ContentsDefinitionGetter: store}.Groups(logger, keys)
		assert.Equal(t, load.SelfReferenceError("foo"), err)
		assert.Nil(t, actual)
	})
}

type mockContentsGetter struct {
	groups map[string]api.ContentsDefinition
	error
}

func (mgg mockContentsGetter) GetContentsDefinition(key *api.GroupKey) (*api.ContentsDefinition, error) {
	g, ok := mgg.groups[key.Key]
	if !ok {
		return nil, mgg.error
	}
	return &g, mgg.error
}
