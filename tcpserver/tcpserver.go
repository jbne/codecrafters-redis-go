package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/respcommands"
	"github.com/codecrafters-io/redis-starter-go/resplib"
	"github.com/lmittmann/tint"
)

type (
	Scan         func() string
	ErrorHandler func(string, bool)
)

func ParseArray(scan <-chan string, handleError ErrorHandler) resplib.RESP2_Array {
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

	ret := make(resplib.RESP2_Array, 0, arrSize)
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
	slog.DebugContext(ctx, "ReadWorker started", "client", remoteAddr)

	err := false
	HandleError := func(str string, terminate bool) {
		_, file, line, _ := runtime.Caller(1)
		slog.ErrorContext(ctx, "Protocol error", "file", file, "line", line, "error", str)
		prefix := "-ERR"
		if terminate {
			err = true
			prefix += "TERM"
		}
		c <- fmt.Sprintf("%s %s\r\n", prefix, str)
	}

	in, ctx := resplib.CreateScannerChannel(ctx, conn, resplib.ScanCRLF)
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "ReadWorker context cancelled", "client", remoteAddr)
			return // Exit when server is shutting down
		default:
		}

		commandArray := ParseArray(in, HandleError)

		if err {
			slog.DebugContext(ctx, "ReadWorker exiting due to protocol error", "client", remoteAddr)
			return
		}

		if len(commandArray) == 0 {
			slog.DebugContext(ctx, "ReadWorker exiting - client disconnected", "client", remoteAddr)
			return // EOF - client disconnected cleanly
		}

		respcommands.ExecuteCommand(ctx, resplib.RESP2_CommandRequest{
			Params:          commandArray,
			ResponseChannel: c,
		})
	}
}

func WriteWorker(ctx context.Context, conn net.Conn, in <-chan string) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr()
	slog.DebugContext(ctx, "WriteWorker started", "client", remoteAddr)
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "WriteWorker context cancelled", "client", remoteAddr)
			return
		case str, ok := <-in:
			if !ok {
				slog.DebugContext(ctx, "WriteWorker exiting - response channel closed", "client", remoteAddr)
				return // Channel closed by ReadWorker
			}

			_, err := writer.WriteString(str)
			if strings.HasPrefix(str, "-ERRTERM") {
				slog.DebugContext(ctx, "WriteWorker exiting - terminating error sent", "client", remoteAddr)
				return
			}

			if err != nil {
				slog.ErrorContext(ctx, "Connection lost", "client", remoteAddr, "error", err)
				break
			}

			slog.DebugContext(ctx, "Response sent", "client", remoteAddr, "response", str)
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

	slog.InfoContext(ctx, "Attempting to start listening", "endpoint", endpoint)
	listener, err := net.Listen(network, endpoint)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to bind", "endpoint", endpoint, "error", err)
		return
	}

	ctx, cancel := context.WithCancel(ctx)

	defer listener.Close()
	in := make(chan net.Conn)
	wg.Go(func() {
		slog.InfoContext(ctx, "Listening for client connections", "endpoint", endpoint)
		for {
			conn, err := listener.Accept()
			if err != nil {
				slog.ErrorContext(ctx, "Error accepting connection", "error", err)
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
			slog.InfoContext(ctx, "Server shutting down")
			cancel()
			return
		case conn := <-in:
			c := make(chan string)
			remoteAddr := conn.RemoteAddr()
			slog.InfoContext(ctx, "Client connected", "client", remoteAddr)
			wg.Go(func() {
				ReadWorker(ctx, conn, c)
				slog.DebugContext(ctx, "ReadWorker done", "client", remoteAddr)
			})
			wg.Go(func() {
				WriteWorker(ctx, conn, c)
				slog.DebugContext(ctx, "WriteWorker done", "client", remoteAddr)
			})
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// Configure colored logging with tint
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04:05.000",
		NoColor:    false,
	})
	slog.SetDefault(slog.New(handler))

	var wg sync.WaitGroup
	wg.Go(func() {
		resplib.ListenStdin(ctx, nil)
		slog.DebugContext(ctx, "StdinWorker done")
		cancel()
	})
	wg.Go(func() {
		ClientConnectionWorker(ctx)
		slog.DebugContext(ctx, "ClientConnectionWorker done")
		cancel()
	})
	wg.Wait()

	slog.DebugContext(ctx, "Clean exit")
}
