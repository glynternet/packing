package load_test

import (
	"log"
	"os"
	"testing"

	"github.com/glynternet/packing/internal/load"
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGroups(t *testing.T) {
	t.Run("empty keys", func(t *testing.T) {
		gs, err := load.Groups(api.GroupKeys{}, nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, gs)
	})

	logger := log.New(os.Stdout, "", log.LstdFlags)

	t.Run("single key missing in getter", func(t *testing.T) {
		logger := log.New(os.Stdout, "", log.LstdFlags)
		keys := api.GroupKeys{Keys: []*api.GroupKey{{Key: "foo"}}}
		store := mockContentsGetter{}
		var expected []list.Group
		gs, err := load.Groups(keys, logger, store)
		assert.NoError(t, err)
		assert.Equal(t, expected, gs)
	})

	t.Run("single key error in getter", func(t *testing.T) {
		keys := api.GroupKeys{Keys: []*api.GroupKey{{Key: "foo"}}}
		expectedErr := errors.New("test error")
		store := mockContentsGetter{error: expectedErr}
		var expected []list.Group
		gs, err := load.Groups(keys, logger, store)
		assert.Equal(t, expectedErr, errors.Cause(err))
		assert.Equal(t, expected, gs)
	})

	t.Run("single key exists", func(t *testing.T) {
		keys := api.GroupKeys{Keys: []*api.GroupKey{{Key: "foo"}}}
		store := mockContentsGetter{
			groups: map[string]api.ContentsDefinition{
				"foo": {Items: &api.Items{Items: []*api.Item{{Name: "fooItem"}}}},
				"bar": {Items: &api.Items{Items: []*api.Item{{Name: "barItem"}}}},
			},
		}
		expected := []list.Group{{
			Name:               "foo",
			ContentsDefinition: api.ContentsDefinition{Items: &api.Items{Items: []*api.Item{{Name: "fooItem"}}}}},
		}
		gs, err := load.Groups(keys, logger, store)
		assert.NoError(t, err)
		assert.Equal(t, expected, gs)
	})

	t.Run("single key containing self group reference", func(t *testing.T) {
		// currently completes but will probably cause a bug when changing the way that groups are loaded
		keys := api.GroupKeys{Keys: []*api.GroupKey{{Key: "foo"}}}
		store := mockContentsGetter{
			groups: map[string]api.ContentsDefinition{
				"foo": {GroupKeys: &api.GroupKeys{Keys: []*api.GroupKey{{Key: "foo"}}}},
			},
		}
		actual, err := load.Groups(keys, logger, store)
		assert.Equal(t, load.GroupSelfReferenceErr, err)
		assert.Nil(t, actual)
	})
}

type mockContentsGetter struct {
	groups map[string]api.ContentsDefinition
	error
}

func (mgg mockContentsGetter) GetContentsDefinition(key api.GroupKey) (*api.ContentsDefinition, error) {
	g, ok := mgg.groups[key.Key]
	if !ok {
		return nil, mgg.error
	}
	return &g, mgg.error
}
