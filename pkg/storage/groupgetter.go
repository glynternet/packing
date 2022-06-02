package storage

import (
	"io"

	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
)

// ContentsDefinitionGetter gets a api.ContentsDefinition
type ContentsDefinitionGetter struct {
	GetReadCloser ReadCloserGetter
	log.Logger
}

// ReadCloserGetter gets an io.ReadCloser for a given reference
type ReadCloserGetter func(ref string) (io.ReadCloser, error)

// GetContentsDefinition gets a Contents for a given reference
func (gg ContentsDefinitionGetter) GetContentsDefinition(ref string) (api.Contents, error) {
	rc, err := gg.GetReadCloser(ref)
	if err != nil {
		return api.Contents{}, errors.Wrapf(err, "getting ReadCloser for ref:%v", ref)
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
			log.ErrorMessage(cErr),
		)
	}()

	def, err := list.ParseContentsDefinition(rc)
	err = errors.Wrapf(err, "loading group for ref:%v", ref)
	return def, err
}
