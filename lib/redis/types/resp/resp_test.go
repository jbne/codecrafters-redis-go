package resptypes

import (
	"testing"
)

func TestParsingCompleteStrings(t *testing.T) {
	tcs := []struct {
		expected string
	}{
		{"+test\r\n"},
		{"+\r\n"},
		{"-Error message\r\n"},
		{"-ERR unknown command 'asdf'\r\n"},
		{"-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"},
		{"-\r\n"},
		{":0\r\n"},
		{":1000\r\n"},
		{"$0\r\n\r\n"},
		{"$-1\r\n"},
		{"$5\r\nhello\r\n"},
		{"$25\r\nwow this actually worked?\r\n"},
		{"*-1\r\n"},
		{"*0\r\n"},
		{"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n"},
		{"*3\r\n:1\r\n:2\r\n:3\r\n"},
		{"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nhello\r\n"},
		{"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n"},
		{"*2\r\n*2\r\n$15\r\n1526985054069-0\r\n*4\r\n$11\r\ntemperature\r\n$2\r\n36\r\n$8\r\nhumidity\r\n$2\r\n95\r\n*2\r\n$15\r\n1526985054079-0\r\n*4\r\n$11\r\ntemperature\r\n$2\r\n37\r\n$8\r\nhumidity\r\n$2\r\n94\r\n"},
		{"*5\r\n$-1\r\n$0\r\n\r\n*-1\r\n*0\r\n*4\r\n$-1\r\n$0\r\n\r\n*-1\r\n*0\r\n"},
	}

	for _, tc := range tcs {
		t.Run(tc.expected, func(t *testing.T) {
			parsed, bytesCount := ParseRespString(tc.expected)
			deserialized := parsed.ToRespString()

			if deserialized != tc.expected {
				t.Errorf("Deserialized does not match! got %q, want %q", deserialized, tc.expected)
			}

			if bytesCount != len(tc.expected) {
				t.Errorf("Byte count does not match! got %d, want %d", bytesCount, len(tc.expected))
			}
		})
	}
}

func TestParsingFailures(t *testing.T) {
	tcs := []struct {
		str string
	}{
		{""},
		{"missing_type_specifier\r\n"},
		{"+missing_delim"},
		{":\r\n"},
		{":not_an_integer\r\n"},
		{"$-2\r\n"},
		{"$4.15\r\ntest\r\n"},
		{"$not_an_integer\r\ntest\r\n"},
		{"$1\r\nincorrect_length\r\n"},
		{"*not_an_integer\r\n"},
		{"*5\r\n"},
		{"*5\r\n$-1\r\n$0\r\n\r\n*-1\r\n*0\r\n*5\r\n$-1\r\n$0\r\n\r\n*-1\r\n*0\r\n"},
	}

	for _, tc := range tcs {
		t.Run(tc.str, func(t *testing.T) {
			parsed, bytesCount := ParseRespString(tc.str)

			if _, ok := parsed.(SimpleError); !ok {
				t.Errorf("Expected SimpleError type! Got: %v", parsed)
			}

			if bytesCount > 0 {
				t.Errorf("Expected byte count to be <= 0! got %d", bytesCount)
			}
		})
	}
}
