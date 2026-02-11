package lib

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/logger"
)

func ScanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func CreateScannerChannel(ctx context.Context, reader io.Reader, splitFunc bufio.SplitFunc) (<-chan string, context.Context) {
	in := make(chan string)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Split(splitFunc)
		defer cancel()
		defer close(in)
		for scanner.Scan() {
			line := scanner.Text()
			logger.DebugContext(ctx, "Scanner read line", "line", line)
			select {
			case <-ctx.Done():
				logger.DebugContext(ctx, "Scanner cancelled by context")
				return
			case in <- line:
			}
		}
		// Check for errors after Scan() returns false
		if err := scanner.Err(); err != nil {
			logger.ErrorContext(ctx, "Scanner error", "error", err)
		}

		logger.DebugContext(ctx, "Scanner channel closed")
	}()

	return in, ctx
}

func StdinWorker(ctx context.Context, out chan<- string) {
	in, ctx := CreateScannerChannel(ctx, os.Stdin, bufio.ScanLines)
	for {
		select {
		case <-ctx.Done():
			logger.DebugContext(ctx, "StdinWorker context cancelled")
			return
		case text := <-in:
			input := strings.TrimSpace(text)
			if input == "" {
				continue
			}

			switch input {
			case "q":
				logger.InfoContext(ctx, "User quit")
				return
			default:
				logger.DebugContext(ctx, "Command parsed", "input", input)

				if out != nil {
					out <- text
				}
			}
		}
	}
}