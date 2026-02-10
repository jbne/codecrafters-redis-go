package utils

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/logger"
)

func StdinWorker(ctx context.Context, out chan<- string) {
	in := CreateScannerChannel(ctx, os.Stdin, bufio.ScanLines)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("StdinWorker context cancelled")
			return
		case text := <-in:
			input := strings.TrimSpace(text)
			if input == "" {
				continue
			}

			switch input {
			case "q":
				logger.Info("User quit")
				return
			default:
				logger.Debug("Command parsed", "input", input)
				out <- text
			}
		}
	}
}

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

func CreateScannerChannel(ctx context.Context, reader io.Reader, splitFunc bufio.SplitFunc) <-chan string {
	in := make(chan string)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Split(splitFunc)
		defer close(in)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				logger.Debug("Scanner cancelled by context")
				return
			case in <- scanner.Text():
			}
		}
		// Check for errors after Scan() returns false
		if err := scanner.Err(); err != nil {
			logger.Error("Scanner error", "error", err)
		}

		logger.Debug("Scanner channel closed")
	}()

	return in
}