package resptypes

import (
	"fmt"
)

type (
	String struct {
		Val string
	}
)

func (r String) ToRespString() string {
	return fmt.Sprintf("+%s\r\n", r.Val)
}

func (r String) ToString() string {
	return r.Val
}
