package resptypes

import (
	"fmt"
)

type (
	SimpleError struct {
		Val error
	}
)

func (r SimpleError) ToRespString() string {
	return fmt.Sprintf("-%s\r\n", r.Val.Error())
}

func (r SimpleError) toString() string {
	return r.Val.Error()
}
