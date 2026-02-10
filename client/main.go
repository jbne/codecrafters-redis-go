package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/logger"
	"github.com/codecrafters-io/redis-starter-go/utils"
)

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

func WriteWorker(ctx context.Context, conn net.Conn, in <-chan string) {
	var buf bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			logger.Debug("WriteWorker context cancelled")
			return
		case input := <-in:
			tokens := RESPify(input)
			if len(tokens) == 0 {
				logger.Warn("No tokens parsed from input", "input", input)
				continue
			}

			buf.Reset()
			fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
			for _, token := range tokens {
				fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
			}

			_, err := conn.Write(buf.Bytes())
			if err != nil {
				logger.Error("Failed to write command", "error", err)
				return
			}

			logger.Debug("Command sent", "request", buf.String())
		}
	}
}

func ReadWorker(ctx context.Context, conn net.Conn) {
	in := utils.CreateScannerChannel(ctx, conn, utils.ScanCRLF)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("ReadWorker context cancelled")
			return
		case line := <-in:
			if len(line) == 0 {
				continue
			}

			// Log the raw RESP response line
			logger.Debug("Response received", "response", line)
		}
	}
}

func main() {
	network := "tcp4"
	address := "localhost"
	port := "6379"
	endpoint := net.JoinHostPort(address, port)

	logger.Info("Connecting to server", "endpoint", endpoint)
	conn, err := net.Dial(network, endpoint)

	if err != nil {
		logger.Error("Failed to connect", "endpoint", endpoint, "error", err)
		return
	}

	logger.Info("Connected to server", "endpoint", endpoint)
	defer conn.Close()

	commandChannel := make(chan string)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Go(func() {
		utils.StdinWorker(ctx, commandChannel)
		logger.Debug("StdinWorker done")
		cancel()
	})
	wg.Go(func() {
		WriteWorker(ctx, conn, commandChannel)
		logger.Debug("WriteWorker done")
	})
	wg.Go(func() {
		ReadWorker(ctx, conn)
		logger.Debug("ReadWorker done")
	})

	wg.Wait()
	logger.Info("Client closed")
}
