package load

// GroupSelfReferenceErr is returned whenever a group references itself
const GroupSelfReferenceErr = loadError("group cannot contain reference to self")

type loadError string

func (sre loadError) Error() string {
	return string(sre)
}
