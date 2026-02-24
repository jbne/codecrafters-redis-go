package resplib

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

func TokenizeCommandLine(input string) []string {
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

	// If we finished but a quote was never closed
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

func SerializeRespArray(arr RESP2_Array) string {
	var sb strings.Builder
	if arr == nil {
		fmt.Fprint(&sb, "*-1\r\n")
	} else {
		fmt.Fprintf(&sb, "*%d\r\n", len(arr))
		for _, str := range arr {
			fmt.Fprintf(&sb, "$%d\r\n%s\r\n", len(str), str)
		}
	}

	return sb.String()
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
			slog.DebugContext(ctx, "Scanner read line", "line", line)
			select {
			case <-ctx.Done():
				slog.DebugContext(ctx, "Scanner cancelled by context")
				return
			case in <- line:
			}
		}
		// Check for errors after Scan() returns false
		if err := scanner.Err(); err != nil {
			slog.ErrorContext(ctx, "Scanner error", "error", err)
		}

		slog.DebugContext(ctx, "Scanner channel closed")
	}()

	return in, ctx
}

func ListenStdin(ctx context.Context, out chan<- string) {
	in, ctx := CreateScannerChannel(ctx, os.Stdin, bufio.ScanLines)
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "StdinWorker context cancelled")
			return
		case text := <-in:
			input := strings.TrimSpace(text)
			if input == "" {
				continue
			}

			switch input {
			case "q":
				slog.InfoContext(ctx, "User quit")
				return
			default:
				slog.DebugContext(ctx, "Command parsed", "input", input)

				if out != nil {
					out <- text
				}
			}
		}
	}
}
