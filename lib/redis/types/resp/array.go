package resptypes

import (
	"fmt"
	"strings"
)

type (
	Array[T RespSerializable] []T
)

var NullArray = Array[RespSerializable](nil)

func (r Array[T]) ToRespString() string {
	if r == nil {
		return "*-1\r\n"
	} else {
		var sb strings.Builder
		fmt.Fprintf(&sb, "*%d\r\n", len(r))
		for _, respType := range r {
			fmt.Fprint(&sb, respType.ToRespString())
		}

		return sb.String()
	}
}
