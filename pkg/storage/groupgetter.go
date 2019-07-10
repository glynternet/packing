package storage

import (
	"io"
	"log"

	"github.com/glynternet/packing/pkg/list"
	"github.com/pkg/errors"
)

type ContentsGetter struct {
	GetReadCloser ReadCloserGetter
	*log.Logger
}

type ReadCloserGetter func(key string) (io.ReadCloser, error)

func (gg ContentsGetter) GetContents(key string) (list.Contents, error) {
	rc, err := gg.GetReadCloser(key)
	if err != nil {
		return list.Contents{}, errors.Wrapf(err, "getting ReadCloser for key:%q", key)
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

	contents, err := LoadContents(rc)
	err = errors.Wrapf(err, "loading group for key:%q", key)
	return contents, err
}
