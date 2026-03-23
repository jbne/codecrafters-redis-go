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

func parseErr(err error) (SimpleError, int) {
	return SimpleError{Val: fmt.Errorf("ERRPARSE %w", err)}, 0
}

func ParseRespString(respStr string) (BaseInterface, int) {
	if respStr == "" {
		return parseErr(fmt.Errorf("Empty RESP string is not valid! Got: %v", respStr))
	}

	nextSeparatorIndex := strings.Index(respStr, "\r\n")
	if nextSeparatorIndex == -1 {
		return parseErr(fmt.Errorf("RESP string must end with CRLF! Got: %v", respStr))
	}

	switch respStr[0] {
	case '+':
		return SimpleString{Val: respStr[1:nextSeparatorIndex]}, nextSeparatorIndex + 2
	case '-':
		return SimpleError{Val: fmt.Errorf("%s", respStr[1:nextSeparatorIndex])}, nextSeparatorIndex + 2
	case ':':
		intValue, err := strconv.ParseInt(respStr[1:nextSeparatorIndex], 10, 64)
		if err != nil {
			return parseErr(fmt.Errorf("Invalid integer value in RESP string! Got: %v", err))
		}
		return Integer{Val: intValue}, nextSeparatorIndex + 2
	case '$':
		if respStr[:nextSeparatorIndex] == "$-1" {
			return NullBulkString, nextSeparatorIndex + 2
		}

		parts := strings.SplitN(respStr[1:], "\r\n", 3)

		length, err := strconv.Atoi(parts[0])
		if err != nil {
			return parseErr(fmt.Errorf("Invalid length value in RESP string! Got: %v", err))
		}

		if length < 0 {
			return parseErr(fmt.Errorf("Length must be positive integer!"))
		}

		if length != len(parts[1]) {
			return parseErr(fmt.Errorf("Bulk string length does not match actual string length! Got: %v, expected: %v", len(parts[1]), length))
		}

		return BulkString{Length: length, Val: parts[1]}, nextSeparatorIndex + len(parts[1]) + 4
	case '*':
		if respStr[:nextSeparatorIndex] == "*-1" {
			return NullArray, nextSeparatorIndex + 2
		}

		expectedLength, err := strconv.Atoi(respStr[1:nextSeparatorIndex])
		if err != nil {
			return parseErr(fmt.Errorf("Invalid length value in RESP array! Err: %v", err))
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
				return parseErr(fmt.Errorf("Could not parse string '%s' to array! Got: %v", respStr, element))
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
			return parseErr(fmt.Errorf("Array length values do not match! Got: %v, expected: %v", len(elements), expectedLength))
		}

		return Array[BaseInterface](elements), totalNumBytes
	default:
		return parseErr(fmt.Errorf("Invalid RESP type prefix! Got: %v", respStr))
	}
}
