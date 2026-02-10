package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lmittmann/tint"
)

type (
	Scan         func() string
	ErrorHandler func(string, bool)

	RESP2_Array          []string
	RESP2_CommandHandler func(RESP2_Array, chan string)
)

var (
	RESP2_Commands_Map = map[string]RESP2_CommandHandler{
		"PING": PING,
		"ECHO": ECHO,
		"SET":  SET,
		"GET":  GET,
	}

	Cache       = map[string]string{}
	CacheMutex  sync.RWMutex
	Timers      = map[string]*time.Timer{}
	TimersMutex sync.Mutex
)

func RespifyArray(tokens []string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
	for _, token := range tokens {
		fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
	}
	return buf.String()
}

func PING(tokens RESP2_Array, c chan string) {
	c <- "+PONG\r\n"
}

func ECHO(tokens RESP2_Array, c chan string) {
	if len(tokens) < 2 {
		c <- "-ERR No message provided to ECHO!\r\n"
		return
	}
	if len(tokens) > 2 {
		c <- "-ERR ECHO accepts exactly 1 argument!\r\n"
		return
	}
	response := tokens[1]
	c <- fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
}

func SET(tokens RESP2_Array, c chan string) {
	arrSize := len(tokens)
	switch {
	case arrSize >= 3:
		key := tokens[1]
		value := tokens[2]

		// Validate key and value are not empty
		if key == "" {
			c <- "-ERR Key cannot be empty!\r\n"
			return
		}
		if value == "" {
			c <- "-ERR Value cannot be empty!\r\n"
			return
		}

		expiryDurationMs := 0
		err := error(nil)
		for i := 3; i < arrSize; i++ {
			if tokens[i] == "PX" {
				if i+1 >= arrSize {
					c <- "-ERR No expiration specified!\r\n"
					return
				} else {
					expiryDurationMs, err = strconv.Atoi(tokens[i+1])
					if err != nil {
						c <- fmt.Sprintf("-ERR Could not convert %s to an int for expiry! Err: %s\r\n", tokens[i+1], err)
						return
					}
				}
			}
		}

		// Stop any existing timer for this key
		TimersMutex.Lock()
		if timer, exists := Timers[key]; exists {
			slog.Debug("Cancelling existing timer", "key", key)
			timer.Stop()
			delete(Timers, key)
		}
		TimersMutex.Unlock()

		CacheMutex.Lock()
		Cache[key] = value
		CacheMutex.Unlock()
		slog.Debug("SET executed", "key", key, "value", value, "expiry_ms", expiryDurationMs)

		if expiryDurationMs > 0 {
			timer := time.NewTimer(time.Millisecond * time.Duration(expiryDurationMs))
			TimersMutex.Lock()
			Timers[key] = timer
			TimersMutex.Unlock()

			go func() {
				<-timer.C

				CacheMutex.Lock()
				delete(Cache, key)
				CacheMutex.Unlock()

				TimersMutex.Lock()
				delete(Timers, key)
				TimersMutex.Unlock()
				slog.Debug("Key expired", "key", key)
			}()
		}

		c <- "+OK\r\n"
	case arrSize == 2:
		c <- fmt.Sprintf("-ERR No value given for key %s!\r\n", tokens[1])
	case arrSize == 1:
		c <- "-ERR No key given!\r\n"
	}
}

func GET(tokens RESP2_Array, c chan string) {
	if len(tokens) < 2 {
		c <- "-ERR No key provided to GET!\r\n"
		return
	}
	key := tokens[1]
	CacheMutex.RLock()
	response, ok := Cache[key]
	CacheMutex.RUnlock()

	if ok {
		slog.Debug("GET cache hit", "key", key)
		c <- fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
	} else {
		slog.Debug("GET cache miss", "key", key)
		c <- "$-1\r\n"
	}
}

func ParseArray(scan <-chan string, handleError ErrorHandler) RESP2_Array {
	line, ok := <-scan
	if !ok {
		return nil // Channel closed - client disconnected
	}
	if line == "" {
		return nil // EOF or empty line - return silently
	}
	if !strings.HasPrefix(line, "*") {
		handleError("ParseArray called on non-array!", true)
		return nil
	}

	arrSize, err := strconv.Atoi(line[1:])
	if err != nil {
		handleError(fmt.Sprintf("Could not extract array size! Error: %v", err), true)
		return nil
	}

	if arrSize < 0 {
		handleError("Array size cannot be negative", true)
		return nil
	}

	ret := make([]string, 0, arrSize)
	for range arrSize {
		line, ok = <-scan
		if !ok {
			handleError("Channel closed while parsing array element", true)
			return nil
		}
		if line == "" {
			handleError("Unexpected empty line while parsing array element", true)
			return nil
		}
		if line[0] != '$' {
			handleError(fmt.Sprintf("Expected bulk string marker '$', got %q", line), true)
			return nil
		}

		// Parse the bulk string length
		bulkLen, err := strconv.Atoi(line[1:])
		if err != nil {
			handleError(fmt.Sprintf("Invalid bulk string length: %v", err), true)
			return nil
		}

		if bulkLen < 0 {
			handleError("Bulk string length cannot be negative", true)
			return nil
		}

		// Read the actual bulk string data
		data, ok := <-scan
		if !ok {
			handleError("Channel closed while reading bulk string data", true)
			return nil
		}
		if len(data) != bulkLen {
			handleError(fmt.Sprintf("Bulk string length mismatch: expected %d bytes, got %d", bulkLen, len(data)), true)
			return nil
		}
		ret = append(ret, data)
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

func ReadWorker(ctx context.Context, conn net.Conn, c chan string) {
	defer close(c)
	remoteAddr := conn.RemoteAddr()
	slog.Debug("ReadWorker started", "client", remoteAddr)

	err := false
	HandleError := func(str string, terminate bool) {
		_, file, line, _ := runtime.Caller(1)
		slog.Error("Protocol error", "file", file, "line", line, "error", str)
		prefix := "-ERR"
		if terminate {
			err = true
			prefix += "TERM"
		}
		c <- fmt.Sprintf("%s %s\r\n", prefix, str)
	}

	in := CreateScannerChannel(ctx, conn, ScanCRLF)

	for {
		select {
		case <-ctx.Done():
			slog.Debug("ReadWorker context cancelled", "client", remoteAddr)
			return // Exit when server is shutting down
		default:
		}

		command := ParseArray(in, HandleError)

		if err {
			slog.Debug("ReadWorker exiting due to protocol error", "client", remoteAddr)
			return
		}

		if len(command) == 0 {
			slog.Debug("ReadWorker exiting - client disconnected", "client", remoteAddr)
			return // EOF - client disconnected cleanly
		}

		respStr := RespifyArray(command)
		slog.Debug("Command received", "client", remoteAddr, "request", respStr)

		respond, ok := RESP2_Commands_Map[command[0]]
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
	slog.Debug("WriteWorker started", "client", remoteAddr)
	writer := bufio.NewWriter(conn)
	for {
		str, ok := <-c
		if !ok {
			slog.Debug("WriteWorker exiting - response channel closed", "client", remoteAddr)
			return // Channel closed by ReadWorker
		}

		slog.Debug("Response sent", "client", remoteAddr, "response", str)
		_, err := writer.WriteString(str)
		if strings.HasPrefix(str, "-ERRTERM") {
			slog.Debug("WriteWorker exiting - terminating error sent", "client", remoteAddr)
			return
		}

		if err != nil {
			slog.Error("Connection lost", "client", remoteAddr, "error", err)
			break
		}

		writer.Flush()
	}
}

func CreateScannerChannel(ctx context.Context, reader io.Reader, splitFunc bufio.SplitFunc) <-chan string {
	in := make(chan string)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Split(splitFunc)
		defer close(in)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				slog.Debug("Scanner cancelled by context")
				return
			case in <- scanner.Text():
			}
		}
		// Check for errors after Scan() returns false
		if err := scanner.Err(); err != nil {
			slog.Error("Scanner error", "error", err)
		}

		slog.Debug("Scanner channel closed")
	}()

	return in
}

func ClientConnectionWorker(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()

	network := "tcp"
	address := "localhost"
	port := "6379"
	endpoint := fmt.Sprintf("%s:%s", address, port)

	slog.Info("Attempting to start listening", "endpoint", endpoint)
	listener, err := net.Listen(network, endpoint)
	if err != nil {
		slog.Error("Failed to bind", "endpoint", endpoint, "error", err)
		return
	}

	in := make(chan net.Conn)
	wg.Go(func() {
		slog.Info("Listening for client connections", "endpoint", endpoint)
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				slog.Error("Error accepting connection", "error", err)
				return
			}

			select {
			case in <- conn:
			case <-ctx.Done():
				conn.Close()
				return
			}
		}
	})

	for {
		select {
		case <-ctx.Done():
			slog.Info("Server shutting down")
			listener.Close() // Unblock the accept goroutine
			return
		case conn := <-in:
			c := make(chan string)
			remoteAddr := conn.RemoteAddr()
			slog.Info("Client connected", "client", remoteAddr)
			wg.Go(func() {
				ReadWorker(ctx, conn, c)
				slog.Debug("ReadWorker done", "client", remoteAddr)
			})
			wg.Go(func() {
				WriteWorker(conn, c)
				slog.Debug("WriteWorker done", "client", remoteAddr)
			})
		}
	}
}

func StdinWorker(ctx context.Context) {
	in := CreateScannerChannel(ctx, os.Stdin, bufio.ScanLines)
	for {
		select {
		case <-ctx.Done():
			return
		case text := <-in:
			slog.Debug("stdin input", "input", text)
			input := strings.TrimSpace(text)
			if input == "" {
				continue
			}

			if input == "q" {
				return
			}
		}
	}
}

func main() {
	// Configure colored logging with tint
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04:05.000",
		NoColor:    false,
	})
	slog.SetDefault(slog.New(handler))

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Go(func() {
		StdinWorker(ctx)
		slog.Info("StdinWorker done")
		cancel()
	})
	wg.Go(func() {
		ClientConnectionWorker(ctx)
		slog.Info("ClientConnectionWorker done")
	})

	wg.Wait()
	slog.Info("Clean exit")
}
