package redislib

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

func ScanResp(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		// Stop scanning if at EOF and no more data to read
		return 0, nil, nil
	}

	if respType, bytesCount := resptypes.ParseRespString(string(data[:])); bytesCount != 0 {
		serialized := respType.ToRespString()
		length := len(serialized)
		return length, data[0:length], nil
	}

	// If bytesCount == 0, then we may have an incomplete message - request more data.
	return 0, nil, nil
}

func CreateScannerChannel(ctx context.Context, cancel context.CancelFunc, reader io.Reader, splitFunc bufio.SplitFunc) <-chan string {
	out := make(chan string)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Split(splitFunc)
		defer cancel()
		defer close(out)
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case <-ctx.Done():
				slog.DebugContext(ctx, "Scanner cancelled by context")
				return
			case out <- line:
				//slog.DebugContext(ctx, "Scanner channel got something", "line", line)
			}
		}
		// Check for errors after Scan() returns false
		if err := scanner.Err(); err != nil {
			slog.ErrorContext(ctx, "Scanner error", "error", err)
			return
		}

		slog.DebugContext(ctx, "Scanner channel closed")
	}()

	return out
}

func ListenStdin(ctx context.Context, cancel context.CancelFunc) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		defer cancel()
		in := CreateScannerChannel(ctx, cancel, os.Stdin, bufio.ScanLines)
		for {
			select {
			case <-ctx.Done():
				slog.DebugContext(ctx, "StdinWorker context cancelled")
				return
			case text := <-in:
				input := strings.TrimSpace(text)
				if input == "" {
					slog.DebugContext(ctx, "Empty stdin line received")
					continue
				}

				switch input {
				case "q":
					slog.InfoContext(ctx, "User quit")
					return
				default:
					slog.DebugContext(ctx, "Got input from stdin", "input", input)

					if out != nil {
						out <- text
					}
				}
			}
		}
	}()
	return out
}
