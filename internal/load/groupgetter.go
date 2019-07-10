package load

import "github.com/glynternet/packing/pkg/list"

type ContentsGetter interface {
	GetContents(string) (list.Contents, error)
}
