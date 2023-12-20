package file

import (
	"io"
	"os"
	"path"

	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

// ReadCloserGetter generates a storage.ReadCloserGetter that returns a io.ReadCloser for a file with the name of the
// api.GroupKey contained within groupsDir
func ReadCloserGetter(groupsDir string) storage.ReadCloserGetter {
	return func(key string) (closer io.ReadCloser, e error) {
		p := path.Join(groupsDir, key)
		f, err := os.Open(p)
		return f, errors.Wrapf(err, "opening file at path:%q", p)
	}
}
