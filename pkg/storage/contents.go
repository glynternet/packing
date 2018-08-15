package storage

import "github.com/glynternet/packing/pkg/list"

type ListContentsGetter interface {
	Get(string) (list.Contents, error)
}
