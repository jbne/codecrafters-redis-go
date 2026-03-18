package resptypes

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	BaseInterface interface {
		ToRespString() string
		toString() string
	}
)

func ToBulkStringArray(tokens []string) Array[BulkString] {
	ret := make(Array[BulkString], len(tokens))
	for i, token := range tokens {
		ret[i] = *NewBulkString(token)
	}

	return ret
}

func ParseRespString(respStr string) (BaseInterface, int) {
	if respStr == "" {
		return SimpleError{Val: fmt.Errorf("ERRPARSE Empty RESP string is not valid! Got: %v", respStr)}, 0
	}

	nextSeparatorIndex := strings.Index(respStr, "\r\n")
	if nextSeparatorIndex == -1 {
		return SimpleError{Val: fmt.Errorf("ERRPARSE RESP string must end with CRLF! Got: %v", respStr)}, 0
	}

	switch respStr[0] {
	case '+':
		return SimpleString{Val: respStr[1:nextSeparatorIndex]}, nextSeparatorIndex + 2
	case '-':
		return SimpleError{Val: fmt.Errorf("%s", respStr[1:nextSeparatorIndex])}, nextSeparatorIndex + 2
	case ':':
		intValue, err := strconv.ParseInt(respStr[1:nextSeparatorIndex], 10, 64)
		if err != nil {
			return SimpleError{Val: fmt.Errorf("ERRPARSE Invalid integer value in RESP string! Got: %v", err)}, 0
		}
		return Integer{Val: intValue}, nextSeparatorIndex + 2
	case '$':
		if respStr[:nextSeparatorIndex] == "$-1" {
			return NullBulkString, nextSeparatorIndex + 2
		}

		parts := strings.SplitN(respStr[1:], "\r\n", 3)

		length, err := strconv.Atoi(parts[0])
		if err != nil {
			return SimpleError{Val: fmt.Errorf("ERRPARSE Invalid length value in RESP string! Got: %v", err)}, 0
		}

		if length < 0 {
			return SimpleError{Val: fmt.Errorf("ERRPARSE Length must be positive integer!")}, 0
		}

		if length != len(parts[1]) {
			return SimpleError{Val: fmt.Errorf("ERRPARSE Bulk string length does not match actual string length! Got: %v, expected: %v", len(parts[1]), length)}, 0
		}

		return BulkString{Length: length, Val: parts[1]}, nextSeparatorIndex + len(parts[1]) + 4
	case '*':
		if respStr[:nextSeparatorIndex] == "*-1" {
			return NullArray, nextSeparatorIndex + 2
		}

		expectedLength, err := strconv.Atoi(respStr[1:nextSeparatorIndex])
		if err != nil {
			return SimpleError{Val: fmt.Errorf("ERRPARSE Invalid length value in RESP array! Err: %v", err)}, 0
		}

		size := 0
		totalNumBytes := nextSeparatorIndex + 2
		elements := make(Array[BaseInterface], expectedLength)
		for size < expectedLength {
			respStr = respStr[nextSeparatorIndex+2:]
			nextSeparatorIndex = strings.Index(respStr, "\r\n")

			if nextSeparatorIndex == -1 {
				break
			}

			element, numBytes := ParseRespString(respStr)
			if numBytes == 0 {
				return SimpleError{Val: fmt.Errorf("ERRPARSE Could not parse string '%s' to array! Got: %v", respStr, element)}, 0
			}

			switch respStr[0] {
			case '$', '*':
				respStr = respStr[numBytes-2:]
				nextSeparatorIndex = strings.Index(respStr, "\r\n")
			}

			totalNumBytes += numBytes
			elements[size] = element
			size = size + 1
		}

		if size != expectedLength {
			return SimpleError{Val: fmt.Errorf("ERRPARSE Array length values do not match! Got: %v, expected: %v", len(elements), expectedLength)}, 0
		}

		return Array[BaseInterface](elements), totalNumBytes
	default:
		return SimpleError{Val: fmt.Errorf("ERRPARSE Invalid RESP type prefix! Got: %v", respStr)}, 0
	}
}
