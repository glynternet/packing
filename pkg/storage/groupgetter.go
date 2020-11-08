package storage

import (
	"io"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
)

// ContentsDefinitionGetter gets a api.ContentsDefinition
type ContentsDefinitionGetter struct {
	GetReadCloser ReadCloserGetter
	log.Logger
}

// ReadCloserGetter gets an io.ReadCloser for a given api.GroupKey
type ReadCloserGetter func(key *api.GroupKey) (io.ReadCloser, error)

// GetContentsDefinition gets a api.ContentsDefinition fir a given api.GroupKey
func (gg ContentsDefinitionGetter) GetContentsDefinition(key *api.GroupKey) (*api.ContentsDefinition, error) {
	rc, err := gg.GetReadCloser(key)
	if err != nil {
		return nil, errors.Wrapf(err, "getting ReadCloser for key:%v", key)
	}

	defer func() {
		cErr := rc.Close()
		if cErr == nil {
			return
		}
		if err == nil {
			err = cErr
			return
		}
		_ = gg.Logger.Log(
			log.Message("Error closing group ReadCloser"),
			log.Error(cErr),
		)
	}()

	def, err := list.ParseContentsDefinition(rc)
	err = errors.Wrapf(err, "loading group for key:%v", key)
	return def, err
}
