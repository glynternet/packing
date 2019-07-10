package load

const GroupSelfReferenceErr = loadError("group cannot contain reference to self")

type loadError string

func (sre loadError) Error() string {
	return string(sre)
}
