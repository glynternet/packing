package load

import "fmt"

type SelfReferenceError string

func (sre SelfReferenceError) Error() string {
	return fmt.Sprintf("group cannot contain reference to self: %s", string(sre))
}
