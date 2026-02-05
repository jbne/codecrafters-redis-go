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
	"strings"
	"sync"
	"time"

	"github.com/lmittmann/tint"
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

func RESPify(input string) []string {
	var ret []string
	var current []rune
	inQuote := false
	quoteIdx := -1 // Track where the active quote started
	escapeNext := false

	for i, r := range input {
		if escapeNext {
			// Handle escaped character
			current = append(current, r)
			escapeNext = false
			continue
		}

		if r == '\\' && inQuote {
			// Next character is escaped
			escapeNext = true
			continue
		}

		if r == '"' {
			// Start of a quoted block
			if !inQuote && (i == 0 || input[i-1] == ' ') {
				inQuote = true
				quoteIdx = i
				continue
			}
			// End of a quoted block
			if inQuote && (i == len(input)-1 || input[i+1] == ' ') {
				inQuote = false
				quoteIdx = -1
				continue
			}
			// Literal quote inside a word
			current = append(current, r)
		} else if r == ' ' && !inQuote {
			if len(current) > 0 {
				ret = append(ret, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}

	// BUG FIX: If we finished but a quote was never closed
	if inQuote {
		// Treat the start-quote as a literal and re-append the rest
		// A simple way is to take the slice from the quoteIdx and split it by fields
		remainder := strings.Fields(input[quoteIdx:])
		ret = append(ret, remainder...)
	} else if len(current) > 0 {
		ret = append(ret, string(current))
	}

	return ret
}

func CreateScannerChannel(reader io.Reader, splitFunc bufio.SplitFunc) <-chan string {
	in := make(chan string)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Split(splitFunc)
		for scanner.Scan() {
			in <- scanner.Text()
		}
		// Check for errors after Scan() returns false
		if err := scanner.Err(); err != nil {
			slog.Error("Scanner error", "error", err)
		}

		slog.Debug("Scanner channel closed")
		close(in)
	}()

	return in
}

func StdinWorker(ctx context.Context, out chan<- string) {
	in := CreateScannerChannel(os.Stdin, bufio.ScanLines)
	for {
		select {
		case <-ctx.Done():
			slog.Debug("StdinWorker context cancelled")
			return
		case text := <-in:
			input := strings.TrimSpace(text)
			if input == "" {
				continue
			}

			switch input {
			case "q":
				slog.Info("User quit")
				return
			default:
				slog.Debug("Command parsed", "input", input)
				out <- text
			}
		}
	}
}

func WriteWorker(ctx context.Context, conn net.Conn, in <-chan string) {
	var buf bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			slog.Debug("WriteWorker context cancelled")
			return
		case input := <-in:
			buf.Reset()

			tokens := RESPify(input)
			if len(tokens) == 0 {
				slog.Warn("No tokens parsed from input", "input", input)
				continue
			}

			fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
			for _, token := range tokens {
				fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
			}

			_, err := conn.Write(buf.Bytes())
			if err != nil {
				slog.Error("Failed to write command", "error", err)
				return
			}

			slog.Debug("Command sent", "request", buf.String())
		}
	}
}

func ReadWorker(ctx context.Context, conn net.Conn) {
	in := CreateScannerChannel(conn, ScanCRLF)
	for {
		select {
		case <-ctx.Done():
			slog.Debug("ReadWorker context cancelled")
			return
		case line := <-in:
			if len(line) == 0 {
				continue
			}

			// Log the raw RESP response line
			slog.Debug("Response received", "response", line)
		}
	}
}

func main() {
	// Configure colored logging with tint
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.DateTime,
		NoColor:    false,
	})
	slog.SetDefault(slog.New(handler))

	network := "tcp4"
	address := "localhost"
	port := "6379"
	endpoint := fmt.Sprintf("%s:%s", address, port)

	slog.Info("Connecting to server", "endpoint", endpoint)
	conn, err := net.Dial(network, endpoint)

	if err != nil {
		slog.Error("Failed to connect", "endpoint", endpoint, "error", err)
		return
	}

	slog.Info("Connected to server", "endpoint", endpoint)
	defer conn.Close()

	commandChannel := make(chan string)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Go(func() {
		StdinWorker(ctx, commandChannel)
		slog.Debug("StdinWorker done")
		cancel()
	})
	wg.Go(func() {
		WriteWorker(ctx, conn, commandChannel)
		slog.Debug("WriteWorker done")
	})
	wg.Go(func() {
		ReadWorker(ctx, conn)
		slog.Debug("ReadWorker done")
	})

	wg.Wait()
	slog.Info("Client closed")
}
