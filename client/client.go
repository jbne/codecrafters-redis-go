package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
)

func ScanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		return i + 2, data[0 : i+2], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func StdinWorker(c chan string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		text := scanner.Text()

		if strings.TrimSpace(text) == "" {
			continue
		}

		c <- text
	}
}

func WriteWorker(conn net.Conn, inputChannel chan string) {
	var buf bytes.Buffer
	for input := range inputChannel {
		if strings.TrimSpace(input) == "" {
			continue
		}

		// 1. Serialize directly into the local buffer
		buf.Reset()
		tokens := strings.Fields(input)
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
	}
}

func ReadWorker(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Split(ScanCRLF)
	for {
		if !scanner.Scan() {
			return
		}
		line := scanner.Text()
		fmt.Fprintf(os.Stdout, "%q\n", line)
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
