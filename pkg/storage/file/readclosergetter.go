package file

import (
	"io"
	"os"
	"path"

	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

func ReadCloserGetter(groupsDir string) storage.ReadCloserGetter {
	return func(key api.GroupKey) (closer io.ReadCloser, e error) {
		p := path.Join(groupsDir, key.Key)
		f, err := os.Open(p)
		return f, errors.Wrapf(err, "opening file at path:%q", p)
	}
}
