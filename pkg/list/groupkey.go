package list

type GroupKey string

type GroupKeys []GroupKey

func (gks GroupKeys) Contains(k GroupKey) bool {
	for _, gk := range gks {
		if gk == k {
			return true
		}
	}
	return false
}
