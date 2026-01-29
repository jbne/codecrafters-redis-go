package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type (
	Scan         func() string
	ErrorHandler func(string, bool)

	RESP2_Array          []string
	RESP2_CommandHandler func(RESP2_Array, chan string)
)

var (
	RESP2_SupportedCommands_Map = map[string]RESP2_CommandHandler{
		"PING": PING,
		"ECHO": ECHO,
		"SET":  SET,
		"GET":  GET,
	}

	Cache = map[string]string{}
)

func PING(tokens RESP2_Array, c chan string) {
	c <- "+PONG\r\n"
}

func ECHO(tokens RESP2_Array, c chan string) {
	response := strings.Join(tokens[1:], " ")
	c <- fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
}

func SET(tokens RESP2_Array, c chan string) {
	arrSize := len(tokens)
	switch {
	case arrSize > 3:
		args := strings.TrimSpace(strings.Join(tokens[2:], " "))
		if args[0] == '"' {
			nextQuoteIndex := strings.Index(args[1:], "\"")
			if nextQuoteIndex < 0 {
				Cache[tokens[1]] = tokens[2]
			} else {
				Cache[tokens[1]] = args[1 : nextQuoteIndex+1]
			}
		} else {
			Cache[tokens[1]] = tokens[2]
		}
		c <- "+OK\r\n"
	case arrSize == 3:
		Cache[tokens[1]] = tokens[2]
		c <- "+OK\r\n"
	case arrSize == 2:
		c <- fmt.Sprintf("-ERR No value given for key %s!\r\n", tokens[1])
	case arrSize == 1:
		c <- "-ERR No key given! %s!\r\n"
	}
}

func GET(tokens RESP2_Array, c chan string) {
	if len(tokens) > 1 {
		response, ok := Cache[tokens[1]]
		if ok {
			c <- fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
		} else {
			c <- "$-1\r\n"
		}
	}
}

func ProcessArray(scan Scan, handleError ErrorHandler) RESP2_Array {
	line := scan()
	if !strings.HasPrefix(line, "*") {
		handleError("ProcessArray called on non-array!", true)
		return nil
	}

	arrSize, err := strconv.Atoi(line[1:])
	if err != nil {
		handleError(fmt.Sprintf("Could not extract array size! Error: %v", err), true)
		return nil
	}

	ret := make([]string, 0)
	for range arrSize {
		line = scan()
		switch line[0] {
		case '$':
			ret = append(ret, scan())
		}
	}

	return ret
}

func ScanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		return i + 2, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func ReadWorker(conn net.Conn, c chan string) {
	remoteAddr := conn.RemoteAddr()
	scanner := bufio.NewScanner(conn)
	scanner.Split(ScanCRLF)

	err := false
	HandleError := func(str string, terminate bool) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("%v:%v: %s\n", file, line, str)
		prefix := "-ERR"
		if terminate {
			err = true
			prefix += "TERM"
		}
		c <- fmt.Sprintf("%s %s\r\n", prefix, str)
	}

	Scan := func() string {
		scanner.Scan()
		line := scanner.Text()
		return line
	}

	for {
		command := ProcessArray(Scan, HandleError)
		fmt.Printf("[%s] Read from %s: %q\n", time.Now().UTC().Format("2006-01-02 15:04:05Z"), remoteAddr, command)
		if err {
			return
		}

		respond, ok := RESP2_SupportedCommands_Map[command[0]]
		if !ok {
			HandleError(fmt.Sprintf("Unrecognized command '%s'!", command[0]), false)
			continue
		}

		respond(command, c)
	}
}

func WriteWorker(conn net.Conn, c chan string) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr()
	writer := bufio.NewWriter(conn)
	for {
		str := string(<-c)
		fmt.Printf("[%s] Writing to %s: %q\n", time.Now().UTC().Format("2006-01-02 15:04:05Z"), remoteAddr, str)
		_, err := writer.WriteString(str)
		if strings.HasPrefix(str, "-ERRTERM") {
			return
		}

		if err != nil {
			fmt.Printf("Connection lost: %v\n", err)
			break
		}

		writer.Flush()
	}
}

func main() {
	network := "tcp"
	address := "localhost"
	port := "6379"
	endpoint := fmt.Sprintf("%s:%s", address, port)

	fmt.Printf("Start listening on %s\n", endpoint)
	listener, err := net.Listen(network, endpoint)
	if err != nil {
		fmt.Printf("Failed to bind to %s: %s", endpoint, err)
		os.Exit(1)
	}

	defer listener.Close()
	for {
		fmt.Println("Waiting for client connections...")
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}

		c := make(chan string)
		fmt.Printf("Client connected! RemoteAddr: %s\n", conn.RemoteAddr())
		go ReadWorker(conn, c)
		go WriteWorker(conn, c)
	}
}
