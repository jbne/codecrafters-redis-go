package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/commands"
	"github.com/codecrafters-io/redis-starter-go/lib"
	"github.com/codecrafters-io/redis-starter-go/logger"
)

type (
	Scan         func() string
	ErrorHandler func(string, bool)
)

func RespifyArray(tokens []string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
	for _, token := range tokens {
		fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
	}
	return buf.String()
}

func ParseArray(scan <-chan string, handleError ErrorHandler) commands.RESP2_Array {
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

func ReadWorker(ctx context.Context, conn net.Conn, c chan string) {
	defer close(c)
	remoteAddr := conn.RemoteAddr()
	logger.DebugContext(ctx, "ReadWorker started", "client", remoteAddr)

	err := false
	HandleError := func(str string, terminate bool) {
		_, file, line, _ := runtime.Caller(1)
		logger.ErrorContext(ctx, "Protocol error", "file", file, "line", line, "error", str)
		prefix := "-ERR"
		if terminate {
			err = true
			prefix += "TERM"
		}
		c <- fmt.Sprintf("%s %s\r\n", prefix, str)
	}

	in, ctx := lib.CreateScannerChannel(ctx, conn, lib.ScanCRLF)

	for {
		select {
		case <-ctx.Done():
			logger.DebugContext(ctx, "ReadWorker context cancelled", "client", remoteAddr)
			return // Exit when server is shutting down
		default:
		}

		commandArray := ParseArray(in, HandleError)

		if err {
			logger.DebugContext(ctx, "ReadWorker exiting due to protocol error", "client", remoteAddr)
			return
		}

		if len(commandArray) == 0 {
			logger.DebugContext(ctx, "ReadWorker exiting - client disconnected", "client", remoteAddr)
			return // EOF - client disconnected cleanly
		}

		respStr := RespifyArray(commandArray)
		logger.DebugContext(ctx, "Command received", "client", remoteAddr, "request", respStr)

		entry, ok := commands.RESP2_Commands_Map[commandArray[0]]
		if !ok {
			HandleError(fmt.Sprintf("Unrecognized command '%s'!", commandArray[0]), false)
			continue
		}

		response := entry.Execute(commands.RESP2_CommandHandlerParams{
			Ctx:    ctx,
			Params: commandArray,
		})
		c <- string(response)
	}
}

func WriteWorker(ctx context.Context, conn net.Conn, in <-chan string) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr()
	logger.DebugContext(ctx, "WriteWorker started", "client", remoteAddr)
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-ctx.Done():
			logger.DebugContext(ctx, "WriteWorker context cancelled", "client", remoteAddr)
			return
		case str, ok := <-in:
			if !ok {
				logger.DebugContext(ctx, "WriteWorker exiting - response channel closed", "client", remoteAddr)
				return // Channel closed by ReadWorker
			}

			logger.DebugContext(ctx, "Response sent", "client", remoteAddr, "response", str)
			_, err := writer.WriteString(str)
			if strings.HasPrefix(str, "-ERRTERM") {
				logger.DebugContext(ctx, "WriteWorker exiting - terminating error sent", "client", remoteAddr)
				return
			}

			if err != nil {
				logger.ErrorContext(ctx, "Connection lost", "client", remoteAddr, "error", err)
				break
			}

			writer.Flush()
		}
	}
}

func ClientConnectionWorker(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()

	network := "tcp"
	address := "localhost"
	port := "6379"
	endpoint := fmt.Sprintf("%s:%s", address, port)

	logger.InfoContext(ctx, "Attempting to start listening", "endpoint", endpoint)
	listener, err := net.Listen(network, endpoint)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to bind", "endpoint", endpoint, "error", err)
		return
	}

	ctx, cancel := context.WithCancel(ctx)

	defer listener.Close()
	in := make(chan net.Conn)
	wg.Go(func() {
		logger.InfoContext(ctx, "Listening for client connections", "endpoint", endpoint)
		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.ErrorContext(ctx, "Error accepting connection", "error", err)
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
			logger.InfoContext(ctx, "Server shutting down")
			cancel()
			return
		case conn := <-in:
			c := make(chan string)
			remoteAddr := conn.RemoteAddr()
			logger.InfoContext(ctx, "Client connected", "client", remoteAddr)
			wg.Go(func() {
				ReadWorker(ctx, conn, c)
				logger.DebugContext(ctx, "ReadWorker done", "client", remoteAddr)
			})
			wg.Go(func() {
				WriteWorker(ctx, conn, c)
				logger.DebugContext(ctx, "WriteWorker done", "client", remoteAddr)
			})
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Go(func() {
		lib.StdinWorker(ctx, nil)
		logger.DebugContext(ctx, "StdinWorker done")
		cancel()
	})
	wg.Go(func() {
		ClientConnectionWorker(ctx)
		logger.DebugContext(ctx, "ClientConnectionWorker done")
		cancel()
	})

	wg.Wait()
	logger.DebugContext(ctx, "Clean exit")
}
