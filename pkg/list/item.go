package list

type Item string

type Items []Item

func (is Items) Contains(i Item) bool {
	for _, ii := range is {
		if ii == i {
			return true
		}
	}
	return false
}
