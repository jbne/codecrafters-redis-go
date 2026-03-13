package resptypes

import (
	"fmt"
	"strconv"
)

type (
	Integer struct {
		Val int64
	}
)

func (r Integer) ToRespString() string {
	return fmt.Sprintf(":%d\r\n", r.Val)
}

func (r Integer) ToString() string {
	return strconv.FormatInt(r.Val, 10)
}
