package resptypes

import (
	"fmt"
	"strings"
)

type (
	Array[T BaseInterface] []T
)

var NullArray = Array[BaseInterface](nil)

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

func (r Array[T]) toString() string {
	var sb strings.Builder

	fmt.Fprint(&sb, "[\r\n")
	for _, e := range r {
		fmt.Fprintf(&sb, "%s\r\n", e.toString())
	}
	fmt.Fprint(&sb, "]")

	return sb.String()
}
