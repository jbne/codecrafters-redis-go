package resptypes

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	BaseType interface {
		ToRespString() string
		ToString() string
	}
)

func FromRespString(respStr string) (result BaseType, ok bool) {
	if respStr == "" {
		return Error{Val: fmt.Errorf("-ERRPARSE Empty RESP string is not valid! Got: %v", respStr)}, false
	}

	nextSeparatorIndex := strings.Index(respStr, "\r\n")
	if nextSeparatorIndex == -1 {
		return Error{Val: fmt.Errorf("-ERRPARSE RESP string must end with CRLF! Got: %v", respStr)}, false
	}

	switch respStr[0] {
	case '+':
		return String{Val: respStr[1:nextSeparatorIndex]}, true
	case '-':
		return Error{Val: fmt.Errorf("%s", respStr[1:nextSeparatorIndex])}, true
	case ':':
		intValue, err := strconv.ParseInt(respStr[1:nextSeparatorIndex], 10, 64)
		if err != nil {
			return Error{Val: fmt.Errorf("-ERRPARSE Invalid integer value in RESP string! Got: %v", err)}, false
		}
		return Integer{Val: intValue}, true
	case '$':
		if respStr[:nextSeparatorIndex+2] == "$-1\r\n" {
			return BulkString{Length: -1}, true
		}

		parts := strings.SplitN(respStr[1:], "\r\n", 3)

		length, err := strconv.Atoi(parts[0])
		if err != nil {
			return Error{Val: fmt.Errorf("-ERRPARSE Invalid length value in RESP string! Got: %v", err)}, false
		}

		if length != len(parts[1]) {
			return Error{Val: fmt.Errorf("-ERRPARSE Bulk string length does not match actual string length! Got: %v, expected: %v", len(parts[1]), length)}, false
		}

		return BulkString{Length: length, Val: parts[1]}, true
	case '*':
		if respStr == "*-1\r\n" {
			return NullArray, true
		}

		expectedLength, err := strconv.Atoi(respStr[1:nextSeparatorIndex])
		if err != nil {
			return Error{Val: fmt.Errorf("-ERRPARSE Invalid length value in RESP array! Err: %v", err)}, false
		}

		size := 0
		elements := make(Array[BaseType], expectedLength)
		for size < expectedLength {
			respStr = respStr[nextSeparatorIndex+2:]
			nextSeparatorIndex = strings.Index(respStr, "\r\n")

			if nextSeparatorIndex == -1 {
				break
			}

			element, ok := FromRespString(respStr)
			if !ok {
				return Error{Val: fmt.Errorf("-ERRPARSE Could not parse string '%s' to array! Got: %v", respStr, element)}, false
			}

			switch respStr[0] {
			case '$', '*':
				elementLength := len(element.ToRespString())
				respStr = respStr[elementLength-2:]
				nextSeparatorIndex = strings.Index(respStr, "\r\n")
			}

			elements[size] = element
			size = size + 1
		}

		if len(elements) != expectedLength {
			return Error{Val: fmt.Errorf("-ERRPARSE Array length values do not match! Got: %v, expected: %v", len(elements), expectedLength)}, false
		}

		return Array[BaseType](elements), true
	default:
		return Error{Val: fmt.Errorf("-ERRPARSE Invalid RESP type prefix! Got: %v", respStr)}, false
	}
}
