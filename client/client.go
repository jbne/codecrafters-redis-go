package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
)

func ScanBufferLength(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Always return the entire buffer that was read
	return len(data), data, nil
}

func StdinWorker(c chan string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		text := scanner.Text()

		if strings.TrimSpace(text) == "" {
			continue
		}

		fmt.Printf("Got %q from stdin\n", text)
		c <- text
	}
}

func RESPify(input string) []string {
	var ret []string
	var current []rune
	inQuote := false
	quoteIdx := -1 // Track where the active quote started

	for i, r := range input {
		if r == '"' {
			// Start of a quoted block
			if !inQuote && (i == 0 || input[i-1] == ' ') {
				inQuote = true
				quoteIdx = i
				continue
			}
			// End of a quoted block
			if inQuote && (i == len(input)-1 || input[i+1] == ' ') {
				inQuote = false
				quoteIdx = -1
				continue
			}
			// Literal quote inside a word
			current = append(current, r)
		} else if r == ' ' && !inQuote {
			if len(current) > 0 {
				ret = append(ret, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}

	// BUG FIX: If we finished but a quote was never closed
	if inQuote {
		// Treat the start-quote as a literal and re-append the rest
		// A simple way is to take the slice from the quoteIdx and split it by fields
		remainder := strings.Fields(input[quoteIdx:])
		ret = append(ret, remainder...)
	} else if len(current) > 0 {
		ret = append(ret, string(current))
	}

	return ret
}

func WriteWorker(conn net.Conn, inputChannel chan string) {
	var buf bytes.Buffer
	for input := range inputChannel {
		if strings.TrimSpace(input) == "" {
			continue
		}

		// 1. Serialize directly into the local buffer
		buf.Reset()

		//tokens := strings.Fields()
		tokens := RESPify(input)
		fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
		for _, token := range tokens {
			fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
		}

		// 2. Write directly to the connection
		_, err := conn.Write(buf.Bytes())
		if err != nil {
			fmt.Printf("Connection lost: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stdout, "Wrote: %q\n", buf.String())
	}
}

func ReadWorker(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Split(ScanBufferLength)
	for {
		if !scanner.Scan() {
			return
		}
		line := scanner.Text()
		fmt.Fprintf(os.Stdout, "Read: %q\n", line)
	}
}

func main() {
	network := "tcp4"
	address := "localhost"
	port := "6379"
	endpoint := fmt.Sprintf("%s:%s", address, port)

	fmt.Printf("Dialing %s\n", endpoint)
	conn, err := net.Dial(network, endpoint)

	if err != nil {
		fmt.Printf("Failed to dial %s\n", endpoint)
		os.Exit(1)
	}

	fmt.Printf("Successfully dialed %s!\n", endpoint)
	defer conn.Close()

	stdinChannel := make(chan string)
	go StdinWorker(stdinChannel)

	writeChannel := make(chan string)
	go WriteWorker(conn, writeChannel)

	go ReadWorker(conn)

	for {
		writeChannel <- <-stdinChannel
	}
}
