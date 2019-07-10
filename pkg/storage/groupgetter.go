package storage

import (
	"io"
	"log"

	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

type ContentsDefinitionGetter struct {
	GetReadCloser ReadCloserGetter
	*log.Logger
}

type ReadCloserGetter func(key list.GroupKey) (io.ReadCloser, error)

func (gg ContentsDefinitionGetter) GetContentsDefinition(key list.GroupKey) (*list.ContentsDefinition, error) {
	rc, err := gg.GetReadCloser(key)
	if err != nil {
		return nil, errors.Wrapf(err, "getting ReadCloser for key:%q", key)
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
		gg.Logger.Println(errors.Wrap(cErr, "closing group ReadCloser"))
	}()

	def, err := list.ParseContentsDefinition(rc)
	err = errors.Wrapf(err, "loading group for key:%q", key)
	return &def, err
}
