package resptypes

import (
	"fmt"
)

type (
	Integer struct {
		Val int64
	}
)

func (r Integer) ToRespString() string {
	return fmt.Sprintf(":%d\r\n", r.Val)
}
