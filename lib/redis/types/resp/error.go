package resptypes

import (
	"fmt"
)

type (
	Error struct {
		Val error
	}
)

func (r Error) ToRespString() string {
	return fmt.Sprintf("-%s\r\n", r.Val.Error())
}

func (r Error) ToString() string {
	return r.Val.Error()
}
