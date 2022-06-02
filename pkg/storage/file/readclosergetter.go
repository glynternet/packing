package file

import (
	"io"
	"os"
	"path"

	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

// ReadCloserGetter generates a storage.ReadCloserGetter that returns an io.ReadCloser for a file with filename of the
// api.Reference key contained within groupsDir.
func ReadCloserGetter(groupsDir string) storage.ReadCloserGetter {
	return func(key string) (closer io.ReadCloser, e error) {
		if key == "" {
			return nil, errors.New("no ref provided")
		}
		p := path.Join(groupsDir, key)
		f, err := os.Open(p)
		return f, errors.Wrapf(err, "opening file at path:%q", p)
	}
}
