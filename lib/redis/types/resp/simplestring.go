package resptypes

import (
	"fmt"
)

type (
	SimpleString struct {
		Val string
	}
)

func (r SimpleString) ToRespString() string {
	return fmt.Sprintf("+%s\r\n", r.Val)
}

func (r SimpleString) toString() string {
	return r.Val
}
