package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/lib/logger"
	redislib "github.com/codecrafters-io/redis-starter-go/lib/redis/common"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

func WriteWorker(ctx context.Context, conn net.Conn, in <-chan string) {
	var buf bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "WriteWorker context cancelled")
			return
		case input := <-in:
			tokens := redislib.TokenizeCommandLine(input)
			if len(tokens) == 0 {
				slog.WarnContext(ctx, "No tokens parsed from input", "input", input)
				continue
			}

			buf.Reset()
			fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
			for _, token := range tokens {
				fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
			}

			_, err := conn.Write(buf.Bytes())
			if err != nil {
				slog.ErrorContext(ctx, "Failed to write command", "error", err)
				return
			}

			slog.DebugContext(ctx, "Command sent", "request", buf.String())
		}
	}
}

func ReadWorker(ctx context.Context, in <-chan string) {
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "ReadWorker context cancelled")
			return
		case respStr := <-in:
			respType, ok := resptypes.FromRespString(respStr)

			if !ok {
				slog.ErrorContext(ctx, "Received invalid RESP response!", "respStr", respStr, "respType", respType)
			}

			slog.DebugContext(ctx, "Response received", "respStr", respStr, "respType", respType)
			fmt.Println(respType.ToString())
		}
	}
}

func main() {
	slog.SetDefault(slog.New(logger.NewHandler(slog.LevelDebug)))
	ctx, cancel := context.WithCancel(context.Background())

	network := "tcp4"
	address := "localhost"
	port := "6379"
	endpoint := net.JoinHostPort(address, port)

	slog.With("endpoint", endpoint).InfoContext(ctx, "Connecting to server")
	conn, err := net.Dial(network, endpoint)

	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect", "error", err)
		return
	}

	slog.InfoContext(ctx, "Connected to server")
	defer conn.Close()

	commandChannel := make(chan string)

	var wg sync.WaitGroup
	wg.Go(func() {
		redislib.ListenStdin(ctx, commandChannel)
		slog.DebugContext(ctx, "StdinWorker done")
		cancel()
	})
	wg.Go(func() {
		WriteWorker(ctx, conn, commandChannel)
		slog.DebugContext(ctx, "WriteWorker done")
		cancel()
	})
	wg.Go(func() {
		in, ctx := redislib.CreateScannerChannel(ctx, conn, redislib.ScanResp)
		ReadWorker(ctx, in)
		slog.DebugContext(ctx, "ReadWorker done")
		cancel()
	})
	wg.Wait()

	slog.InfoContext(ctx, "Client closed")
}
