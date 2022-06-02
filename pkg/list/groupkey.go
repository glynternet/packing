package list

// References is a set of references
type References []string

// Contains returns true if the References contain the given reference
func (rs References) Contains(k string) bool {
	for _, gk := range rs {
		if gk == k {
			return true
		}
	}
	return false
}
