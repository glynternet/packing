package file

import (
	"io"
	"os"
	"path"

	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

func ReadCloserGetter(groupsDir string) storage.ReadCloserGetter {
	return func(key string) (closer io.ReadCloser, e error) {
		p := path.Join(string(groupsDir), key)
		f, err := os.Open(p)
		return f, errors.Wrapf(err, "opening file at path:%q", p)
	}
}
