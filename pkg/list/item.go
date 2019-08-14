package list

import api "github.com/glynternet/packing/pkg/api/build"

type Item string

func ExtractItem(item api.Item) Item {
	return Item(item.Name)
}

type Items []Item

func ExtractItems(items api.Items) Items {
	var is Items
	for _, i := range items.Items {
		is = append(is, ExtractItem(*i))
	}
	return is
}

func (is Items) Contains(i Item) bool {
	for _, ii := range is {
		if ii == i {
			return true
		}
	}
	return false
}
