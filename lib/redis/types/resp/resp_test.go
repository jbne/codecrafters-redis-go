package resptypes

import (
	"testing"
)

func TestParsing(t *testing.T) {
	passingTestCases := []struct {
		expected string
	}{
		{"+test\r\n"},
		{"+\r\n"},
		{"-Error message\r\n"},
		{"-ERR unknown command 'asdf'\r\n"},
		{"-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"},
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
	}

	for _, tt := range passingTestCases {
		t.Run(tt.expected, func(t *testing.T) {
			parsed, _ := FromRespString(tt.expected)
			deserialized := parsed.ToRespString()
			if deserialized != tt.expected {
				t.Errorf("got %q, want %q", deserialized, tt.expected)
			}
		})
	}
}
