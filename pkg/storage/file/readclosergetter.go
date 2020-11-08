package file

import (
	"io"
	"os"
	"path"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

// ReadCloserGetter generates a storage.ReadCloserGetter that returns a io.ReadCloser for a file with the name of the
// api.GroupKey contained within groupsDir
func ReadCloserGetter(groupsDir string) storage.ReadCloserGetter {
	return func(key *api.GroupKey) (closer io.ReadCloser, e error) {
		if key == nil {
			return nil, errors.New("no key provided")
		}
		p := path.Join(groupsDir, key.Key)
		f, err := os.Open(p)
		return f, errors.Wrapf(err, "opening file at path:%q", p)
	}
}
