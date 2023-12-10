package list

// Items is a set of api.Item
type Items []string

// Contains returns true if the Items contains the given api.Item
func (is Items) Contains(i string) bool {
	for _, ii := range is {
		if i == ii {
			return true
		}
	}
	return false
}
