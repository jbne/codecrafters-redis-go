package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/lib/logger"
	rediscommon "github.com/codecrafters-io/redis-starter-go/lib/redis/common"
	redisserverlib "github.com/codecrafters-io/redis-starter-go/lib/redis/server"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

// func ParseArray(ctx context.Context, scan <-chan string, c chan string) <-chan redislib.StringArray {
// 	handleError := func(str string, terminate bool) {
// 		_, file, line, _ := runtime.Caller(1)
// 		slog.ErrorContext(ctx, "Protocol error", "file", file, "line", line, "error", str)
// 		prefix := "-ERR"
// 		if terminate {
// 			prefix += "TERM"
// 		}
// 		c <- fmt.Sprintf("%s %s\r\n", prefix, str)
// 	}
// 	ch := make(chan redistypes.StringArray)
// 	go func() {
// 		line, ok := <-scan
// 		if !ok {
// 			return // Channel closed - client disconnected
// 		}
// 		if line == "" {
// 			return // EOF or empty line - return silently
// 		}
// 		if !strings.HasPrefix(line, "*") {
// 			handleError("ParseArray called on non-array!", true)
// 			return
// 		}

// 		arrSize, err := strconv.Atoi(line[1:])
// 		if err != nil {
// 			handleError(fmt.Sprintf("Could not extract array size! Error: %v", err), true)
// 			return
// 		}

// 		if arrSize < 0 {
// 			handleError("Array size cannot be negative", true)
// 			return
// 		}

// 		ret := make(redislib.StringArray, 0, arrSize)
// 		for range arrSize {
// 			line, ok = <-scan
// 			if !ok {
// 				handleError("Channel closed while parsing array element", true)
// 				return
// 			}
// 			if line == "" {
// 				handleError("Unexpected empty line while parsing array element", true)
// 				return
// 			}
// 			if line[0] != '$' {
// 				handleError(fmt.Sprintf("Expected bulk string marker '$', got %q", line), true)
// 				return
// 			}

// 			// Parse the bulk string length
// 			bulkLen, err := strconv.Atoi(line[1:])
// 			if err != nil {
// 				handleError(fmt.Sprintf("Invalid bulk string length: %v", err), true)
// 				return
// 			}

// 			if bulkLen < 0 {
// 				handleError("Bulk string length cannot be negative", true)
// 				return
// 			}

// 			// Read the actual bulk string data
// 			data, ok := <-scan
// 			if !ok {
// 				handleError("Channel closed while reading bulk string data", true)
// 				return
// 			}
// 			if len(data) != bulkLen {
// 				handleError(fmt.Sprintf("Bulk string length mismatch: expected %d bytes, got %d", bulkLen, len(data)), true)
// 				return
// 			}
// 			ret = append(ret, data)
// 		}
// 		ch <- ret
// 	}()
// 	return ch
// }

func ReadWorker(ctx context.Context, conn net.Conn, c chan string, commandProcessor redisserverlib.CommandProcessor) {
	defer close(c)
	remoteAddr := conn.RemoteAddr()
	slog.DebugContext(ctx, "ReadWorker started", "client", remoteAddr)

	var wg sync.WaitGroup
	defer wg.Wait()
	in, ctx := rediscommon.CreateScannerChannel(ctx, conn, rediscommon.ScanResp)
	requestId := 0
	for {
		ctx := context.WithValue(ctx, logger.RequestIdKey, requestId)
		requestId++

		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "ReadWorker context cancelled")
			return
		case text, ok := <-in:
			if !ok {
				slog.DebugContext(ctx, "ReadWorker exiting due to protocol error")
				return
			}

			wg.Go(func() {
				result := commandProcessor.ExecuteCommand(ctx, text)
				switch result.(type) {
				case resptypes.Null:
					return
				}

				c <- result.ToRespString()
			})
		}
	}
}

func WriteWorker(ctx context.Context, conn net.Conn, in <-chan string) {
	defer conn.Close()
	slog.DebugContext(ctx, "WriteWorker started")
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "WriteWorker context cancelled")
			return
		case str, ok := <-in:
			if !ok {
				slog.DebugContext(ctx, "WriteWorker exiting - response channel closed")
				return // Channel closed by ReadWorker
			}

			_, err := writer.WriteString(str)
			if strings.HasPrefix(str, "-ERRTERM") {
				slog.DebugContext(ctx, "WriteWorker exiting - terminating error sent")
				return
			}

			if err != nil {
				slog.ErrorContext(ctx, "Connection lost", "error", err)
				break
			}

			slog.DebugContext(ctx, "Response sent", "response", str)
			writer.Flush()
		}
	}
}

func ListenConn(ctx context.Context) {
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

	commandProcessor := redisserverlib.NewRedisCommandProcessor()

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

	clientId := 0
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Server shutting down")
			cancel()
			return
		case conn := <-in:
			remoteAddr := conn.RemoteAddr()
			ctx := context.WithValue(ctx, logger.ClientIdKey, clientId)
			c := make(chan string)
			slog.InfoContext(ctx, "Client connected", "client", remoteAddr)
			wg.Go(func() {
				ReadWorker(ctx, conn, c, commandProcessor)
				slog.DebugContext(ctx, "ReadWorker done")
			})
			wg.Go(func() {
				WriteWorker(ctx, conn, c)
				slog.DebugContext(ctx, "WriteWorker done")
			})
			clientId++
		}
	}
}

func main() {
	slog.SetDefault(slog.New(logger.NewHandler(slog.LevelDebug)))
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Go(func() {
		rediscommon.ListenStdin(ctx, nil)
		slog.DebugContext(ctx, "StdinWorker done")
		cancel()
	})
	wg.Go(func() {
		ListenConn(ctx)
		slog.DebugContext(ctx, "ListenConn done")
		cancel()
	})
	wg.Wait()

	slog.DebugContext(ctx, "Clean exit")
}
