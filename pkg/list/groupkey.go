package list

// GroupKeys is a set of api.GroupKey
type GroupKeys []string

// Contains returns true if the GroupKeys contain the given api.GroupKey
func (gks GroupKeys) Contains(k string) bool {
	for _, gk := range gks {
		if gk == k {
			return true
		}
	}
	return false
}
