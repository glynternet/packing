package load

import (
	"log"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/pkg/errors"
)

func GroupsRecursively(keys []string, logger *log.Logger, cg storage.ListContentsGetter, groups map[string]list.Group) error {
	if len(keys) == 0 {
		return nil
	}
	var sublistKeys []string
	for _, key := range keys {
		if _, ok := groups[key]; ok {
			// skip if exists
			continue
		}

		c, err := cg.Get(key)
		if err != nil {
			return errors.Wrapf(err, "getting contents for key:%v", key)
		}

		if len(c.Items) > 0 {
			groups[key] = list.Group{
				Name:  key,
				Items: c.Items,
			}
		}
		sublistKeys = append(sublistKeys, c.SublistKeys...)
	}
	return GroupsRecursively(sublistKeys, logger, cg, groups)
}
