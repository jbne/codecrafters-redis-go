package resptypes

import (
	"fmt"
)

type (
	BulkString struct {
		Length int
		Val    string
	}
)

var NullBulkString = BulkString{Length: -1}

func NewBulkString(str string) *BulkString {
	return &BulkString{
		Length: len(str),
		Val:    str,
	}
}

func (r BulkString) ToRespString() string {
	if r.Length < 0 {
		return "$-1\r\n"
	}

	return fmt.Sprintf("$%d\r\n%s\r\n", r.Length, r.Val)
}

func (r BulkString) toString() string {
	return r.Val
}
