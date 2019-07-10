package file

import (
	"io"
	"os"
	"path"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

func ReadCloserGetter(groupsDir string) storage.ReadCloserGetter {
	return func(key list.GroupKey) (closer io.ReadCloser, e error) {
		p := path.Join(string(groupsDir), string(key))
		f, err := os.Open(p)
		return f, errors.Wrapf(err, "opening file at path:%q", p)
	}
}
